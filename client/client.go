package main

import (
	"context"
	"flag"
	"io"
	"os"
	"os/exec"

	pb "grpc/message"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/examples/data"

	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("scheduler")

var (
	tls                = flag.Bool("tls", false, "Connection uses TLS if true, else plain TCP")
	caFile             = flag.String("ca_file", "", "The file containing the CA root cert file")
	serverAddr         = flag.String("addr", "35.164.231.144:50051", "The server address in the format of host:port")
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
	for {
		data, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}

		log.Infof("DeviceId: %q, Key: %q, Value: %v", data.GetDeviceId(), data.GetKey(), data.GetValue())

		if data.GetKey() == "state" {
			onOff := data.GetValue() == "on"
			sendToChipTool(data.GetDeviceId(), onOff)
		}
	}
}

// TODO: check if it's possible to actually run multiple instances of chip-tool at the same time
func sendToChipTool(deviceId string, onOff bool) {
	// run in a goroutine so we don't block the main thread
	go func() {
		// log.Infof("Sending to chip tool: %v with state %b", deviceId, onOff)

		// run the bash command test.sh
		// Define the path to the bash script
		scriptPath := "./test.sh"

		// convert onOff to a string of either "1" or "0"
		val := "0"
		if onOff {
			val = "1"
		}

		// Create the command to run the script, passing in the args deviceId and onOff
		// NOTE: command line args only accept strings
		cmd := exec.Command(scriptPath, deviceId, val)

		// Run the command and capture its output in real time
		// output, err := cmd.CombinedOutput()
		// if err != nil {
		// 	log.Errorf("Failed to run script: %s\n", err)
		// } else {
		// 	log.Infof("Script output: %s\n", output)
		// }

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			log.Info("Error creating StdoutPipe: ", err)
			return
		}

		stderr, err := cmd.StderrPipe()
		if err != nil {
			log.Info("Error creating StderrPipe: ", err)
			return
		}

		if err := cmd.Start(); err != nil {
			log.Info("Error starting command: ", err)
			return
		}

		go func() {
			// Print stdout in real-time
			io.Copy(os.Stdout, stdout)
		}()

		go func() {
			// Print stderr in real-time
			io.Copy(os.Stderr, stderr)
		}()

		if err := cmd.Wait(); err != nil {
			log.Info("Command finished with error: ", err)
			return
		}

		// log.Info("Command finished successfully")
	}()
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

	conn, err := grpc.Dial(*serverAddr, opts...)
	if err != nil {
		log.Errorf("fail to dial: %v", err)
	}
	defer conn.Close()
	client := pb.NewMessageClient(conn)

	// Feature missing.
	startChat(client, &pb.ClientInit{Id: "myclientid"})
}
