package main

import (
	"context"
	"flag"
	"io"
	"os"
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
	tls                = flag.Bool("tls", false, "Connection uses TLS if true, else plain TCP")
	caFile             = flag.String("ca_file", "", "The file containing the CA root cert file")
	serverAddr         = flag.String("addr", "35.164.231.144:50051", "The server address in the format of host:port")
	serverHostOverride = flag.String("server_host_override", "x.test.example.com", "The server name used to verify the hostname returned by the TLS handshake")
)

func startChat(client pb.MessageClient, clientInit *pb.ClientInit) {
	log.Noticef("Sending initial client data to server %v", clientInit)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	stream, err := client.MessageChat(ctx, clientInit)
	if err != nil {
		log.Errorf("client.MessageChat failed: %v", err)
	}
	for {
		data, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("client.MessageChat failed: %v", err)
		}
		log.Infof("DeviceId: %q, Key: %q, Value: %v", data.GetDeviceId(), data.GetKey(), data.GetValue())
	}
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
