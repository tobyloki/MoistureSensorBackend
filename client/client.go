package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	pb "grpc/message"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/examples/data"

	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("scheduler")

var (
	tls        = flag.Bool("tls", false, "Connection uses TLS if true, else plain TCP")
	caFile     = flag.String("ca_file", "", "The file containing the CA root cert file")
	serverAddr = flag.String("addr", "35.164.231.144:50051", "The server address in the format of host:port")
	// serverAddr         = flag.String("addr", "localhost:50051", "The server address in the format of host:port")
	serverHostOverride = flag.String("server_host_override", "x.test.example.com", "The server name used to verify the hostname returned by the TLS handshake")
)

func startChat(client pb.MessageClient, clientInit *pb.ClientInit) {
	log.Noticef("Sending initial client data to server %v", clientInit)
	// ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stream, err := client.MessageChat(ctx, clientInit)
	if err != nil {
		log.Errorf("client.MessageChat failed: %v", err)
	}
	log.Noticef("Connected to server")

	counter := 0

	var lastData *pb.Data

	for {
		data, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Errorf("Failed to receive a message from server: %v", err)

			// check if code is Unavailable
			if strings.Contains(err.Error(), "Unavailable") {
				counter++
			}

			if lastData != nil {
				data = lastData

				// if data.Value was on, set it to off
				// assuming that the message was intended to invert the value
				if data.GetValue() == "on" {
					data.Value = "off"
				} else {
					data.Value = "on"
				}
			}
		} else {
			counter = 0
		}

		if counter == 2 {
			log.Errorf("Server is unavailable. Exiting...")
			break
		}

		if data == nil {
			continue
		}

		log.Infof("DeviceId: %q, Key: %q, Value: %v", data.GetDeviceId(), data.GetKey(), data.GetValue())

		if data.GetKey() == "state" {
			onOff := data.GetValue() == "on"

			// try to send to chip-tool 3 times before giving up
			for i := 0; i < 3; i++ {
				err := sendToChipTool(data.GetDeviceId(), onOff)
				if err == nil {
					break
				}
			}
		}

		// if error, assume the messages was intended to flip value
		lastData = data
	}
}

// TODO: check if it's possible to actually run multiple instances of chip-tool at the same time
func sendToChipTool(deviceId string, onOff bool) error {
	// run in a goroutine so we don't block the main thread
	// go func() error {

	// run the bash command
	// Define the path to the bash script
	scriptPath := "./script.sh"

	// convert onOff to a string of either "1" or "0"
	state := "0"
	if onOff {
		state = "1"
	}

	log.Infof("Sending to chip tool: %v with state %s", deviceId, state)

	/**********************/

	expiration := 8

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(expiration)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, scriptPath, deviceId, state)

	// Start the command
	// if err := cmd.Start(); err != nil {
	// 	log.Errorf("Failed to start command: %s\n", err)
	// 	return nil, err
	// }

	// Wait for the command to complete or the context to expire
	done := make(chan error, 1)
	output := make([]byte, 0)
	go func() {
		var err error
		output, err = cmd.CombinedOutput()
		done <- err
		close(done)
	}()

	select {
	case err := <-done:
		if err != nil {
			// delete temp-$deviceId.txt
			err = os.Remove("temp-" + deviceId + ".txt")
			if err != nil {
				// log.Error(err)
			}

			if ctx.Err() == context.DeadlineExceeded {
				// If the context has expired, try to kill the process
				if err := cmd.Process.Kill(); err != nil {
					log.Errorf("Failed to kill process: %s\n", err)
				}
				return fmt.Errorf("Command timed out after %d seconds", expiration)
			}
			log.Errorf("Command failed: %s\n", err)
			return err
		}
		// Process the output
		// split the output in array by new line
		outputArray := strings.Split(string(output), "\n")
		for _, line := range outputArray {
			log.Info(line)
			// if line contains status:, split it by space and get the last element
			if strings.Contains(line, "status:") {
				status := strings.Split(line, " ")[len(strings.Split(line, " "))-1]
				if status != "0x0" {
					// throw error
					return fmt.Errorf("Command failed with status: %s\n", status)
				}
			}
		}

		return nil
	case <-ctx.Done():
		// delete temp-$deviceId.txt
		err := os.Remove("temp-" + deviceId + ".txt")
		if err != nil {
			// log.Error(err)
		}

		// If the context has expired, try to kill the process
		if err = cmd.Process.Kill(); err != nil {
			log.Errorf("Failed to kill process: %s\n", err)
		}
		return fmt.Errorf("Command timed out after %d seconds", expiration)
	}

	/**********************/
	// }()
}

func main() {
	// set up logging
	format := logging.MustStringFormatter(
		`%{color}%{time:15:04:05.000} %{shortfunc} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}`,
	)
	loggerBackend := logging.NewLogBackend(os.Stderr, "", 0)
	loggerBackendFormatter := logging.NewBackendFormatter(loggerBackend, format)
	logging.SetBackend(loggerBackendFormatter)

	flag.Parse()
	var opts []grpc.DialOption
	if *tls {
		if *caFile == "" {
			*caFile = data.Path("x509/ca_cert.pem")
		}
		creds, err := credentials.NewClientTLSFromFile(*caFile, *serverHostOverride)
		if err != nil {
			log.Errorf("Failed to create TLS credentials: %v", err)
		}
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	for {
		conn, err := grpc.Dial(*serverAddr, opts...)
		if err != nil {
			log.Errorf("fail to dial: %v", err)
		}
		defer conn.Close()
		client := pb.NewMessageClient(conn)

		startChat(client, &pb.ClientInit{Id: "myclientid"})

		// run a loop every 5 seconds
		// for {
		// 	sendToChipTool("13", true)
		// 	time.Sleep(20 * time.Second)

		// 	sendToChipTool("13", false)
		// 	time.Sleep(20 * time.Second)
		// }

		log.Notice("Connection closed by server. Reconnecting now...")
	}
}
