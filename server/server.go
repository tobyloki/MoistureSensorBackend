package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"time"

	"google.golang.org/grpc"

	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/examples/data"

	pb "grpc/message"

	"google.golang.org/grpc/reflection"

	"github.com/joho/godotenv"
	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("scheduler")

var (
	tls      = flag.Bool("tls", false, "Connection uses TLS if true, else plain TCP")
	certFile = flag.String("cert_file", "", "The TLS cert file")
	keyFile  = flag.String("key_file", "", "The TLS key file")
	port     = flag.Int("port", 50051, "The server port")
)

type messageServer struct {
	pb.UnimplementedMessageServer
}

// ListFeatures lists all features contained within the given bounding Rectangle.
func (s *messageServer) MessageChat(clientInit *pb.ClientInit, stream pb.Message_MessageChatServer) error {
	log.Noticef("New client connected with id: %s", clientInit.GetId())

	// send Message.Data to client every 1 second
	for {
		// key :=

		data := &pb.Data{
			DeviceId: "my device id",
			Key:      clientInit.GetId() + " - mykey",
			Value:    5,
		}
		if err := stream.Send(data); err != nil {
			log.Noticef("Client %s disconnected: %s", clientInit.GetId(), err)
			return err
		}
		time.Sleep(1 * time.Second)
	}
}

func newServer() *messageServer {
	s := &messageServer{}
	return s
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

	// load .env file
	err := godotenv.Load()
	if err != nil {
		log.Warning("Failed to load .env: ", err)
	}
	env := os.Getenv("ENV")
	log.Info("ENV:", string(env))

	serverAddress := fmt.Sprintf(":%d", *port)
	if env == "dev" {
		serverAddress = fmt.Sprintf("localhost:%d", *port)
	}

	lis, err := net.Listen("tcp", serverAddress)
	if err != nil {
		log.Error("failed to listen: %v", err)
	}
	var opts []grpc.ServerOption
	if *tls {
		if *certFile == "" {
			*certFile = data.Path("x509/server_cert.pem")
		}
		if *keyFile == "" {
			*keyFile = data.Path("x509/server_key.pem")
		}
		creds, err := credentials.NewServerTLSFromFile(*certFile, *keyFile)
		if err != nil {
			log.Fatalf("Failed to generate credentials: %v", err)
		}
		opts = []grpc.ServerOption{grpc.Creds(creds)}
	}
	grpcServer := grpc.NewServer(opts...)
	pb.RegisterMessageServer(grpcServer, newServer())

	// Register reflection service on gRPC server.
	reflection.Register(grpcServer)

	// print out the server address
	log.Info("Server listening on port", *port)

	grpcServer.Serve(lis) // the program stops at this line and won't continue anymore
}
