package main

import (
	"encoding/json"
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

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
)

var log = logging.MustGetLogger("scheduler")

var (
	tls      = flag.Bool("tls", false, "Connection uses TLS if true, else plain TCP")
	certFile = flag.String("cert_file", "", "The TLS cert file")
	keyFile  = flag.String("key_file", "", "The TLS key file")
	port     = flag.Int("port", 50051, "The server port")

	rcvQueue      = flag.String("q", "MoistureSensorUpdateActuator", "The name of the queue")
	timeout       = flag.Int64("t", 5, "How long, in seconds, that the message is hidden from others")
	messageHandle = flag.String("m", "", "The receipt handle of the message")

	// 1. Create a channel for the input, with a buffer size of 100.
	input = make(chan UpdateActuatorMessage, 100)
	// 2. Create a multiplexer, with the input channel as the argument.
	//   The multiplexer will listen to the input channel and distribute the messages to the subscribers.
	mux = NewMultiplexer(input)
)

type messageServer struct {
	pb.UnimplementedMessageServer
	sess *session.Session
}

// MessageChat returns a stream of messages to client
func (s *messageServer) MessageChat(clientInit *pb.ClientInit, stream pb.Message_MessageChatServer) error {
	log.Noticef("New client connected with id: %s", clientInit.GetId())

	output := make(chan UpdateActuatorMessage, 100)
	mux.Subscribe(output)

	for message := range output {
		// handle the message
		data := &pb.Data{
			DeviceId: message.ActuatorId,
			Key:      message.Key,
			Value:    message.Value,
		}
		if err := stream.Send(data); err != nil {
			// delay 1 second
			time.Sleep(1 * time.Second)

			// try again
			if err2 := stream.Send(data); err2 != nil {
				log.Noticef("Client [%s] disconnected: %s", clientInit.GetId(), err2)

				mux.Unsubscribe(output)
				return err2
			}
		}
	}

	mux.Unsubscribe(output)
	return nil
}

func newServer(sess *session.Session) *messageServer {
	s := &messageServer{sess: sess}
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

	// setup session for aws
	options := session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Config:            aws.Config{Region: aws.String("us-west-2")},
	}
	if env == "dev" || env == "docker" {
		options.Profile = "aws-osuapp"
	}
	sess := session.Must(session.NewSessionWithOptions(options))

	// set up polling queue
	pollSqs(env, sess)

	// setup grpc server
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
	pb.RegisterMessageServer(grpcServer, newServer(sess))

	// Register reflection service on gRPC server.
	reflection.Register(grpcServer)

	// print out the server address
	log.Info("Server listening on port", *port)

	grpcServer.Serve(lis) // the program stops at this line and won't continue anymore
}

func pollSqs(env string, sess *session.Session) error {
	svc := sqs.New(sess)

	// Get URL of queue
	rcvUrlResult, err := GetQueueURL(svc, rcvQueue)
	if err != nil {
		log.Error("Error getting the queue URL:", err)
		return err
	}

	rcvQueueURL := rcvUrlResult.QueueUrl

	// run in background thread (go routine)
	go func() {
		// look forever, waiting 1 second between checks
		skipSleepFirstRound := true
		for {
			// skip sleep the first round
			if skipSleepFirstRound {
				skipSleepFirstRound = false
			} else {
				// sleep
				time.Sleep(10 * time.Second)
				// log.Info("Waiting...")
			}

			msgResult, err := GetMessages(svc, rcvQueueURL, timeout)
			if err != nil {
				log.Error("Error receiving messages:", err)
				continue
			}

			if len(msgResult.Messages) == 0 {
				log.Notice("No messages received")
				continue
			}

			log.Notice("Received", len(msgResult.Messages), "new message(s). Sending to client...")

			for _, msg := range msgResult.Messages {
				// log.Info("Message ID:     " + *msgResult.Messages[0].MessageId)
				// log.Info("Message Handle: " + *msgResult.Messages[0].ReceiptHandle)
				log.Info(*msg.Body)

				// save the body into a variable called rawMsg
				rawMsg := *msg.Body
				// convert to json
				message := UpdateActuatorMessage{"", "", ""}
				err = json.Unmarshal([]byte(rawMsg), &message)
				if err != nil {
					log.Errorf("Error unmarshalling msg [%s], err: %v", rawMsg, err)
					continue
				}

				// handle the message
				input <- message
				log.Info("Message sent to clients.")

				// delete message
				flag.Set("m", *msg.ReceiptHandle)
				if messageHandle == nil {
					log.Error("No message handle. Can't delete message from queue.")
					continue
				}

				err = DeleteMessage(svc, rcvQueueURL, messageHandle)
				if err != nil {
					log.Error("Error deleting the message:", err)
					continue
				}
			}
		}
	}()

	return nil
}
