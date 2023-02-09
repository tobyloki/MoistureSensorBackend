package main

// note fmt doesn't work in Docker
import (
	"encoding/json"
	"errors"
	"flag"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/joho/godotenv"
	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("scheduler")

const MINUTE = "MINUTE"
const HOUR = "HOUR"
const DAY = "DAY"

// create enum for granularityUnit of minute, hour, day from json
type GranularityUnit string

const (
	Minute GranularityUnit = MINUTE
	Hour   GranularityUnit = HOUR
	Day    GranularityUnit = DAY
)

type RcvMessage struct {
	// name of the key must be capitalized to be exported
	ActuatorId       string          `json:"actuatorId"`
	GranularityValue int             `json:"granularityValue"`
	GranularityUnit  GranularityUnit `json:"granularityUnit"`
}

type SendMessage struct {
	ActuatorId string `json:"actuatorId"`
	State      bool   `json:"state"`
}

func main() {
	format := logging.MustStringFormatter(
		`%{color}%{time:15:04:05.000} %{shortfunc} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}`,
	)
	loggerBackend := logging.NewLogBackend(os.Stderr, "", 0)
	loggerBackendFormatter := logging.NewBackendFormatter(loggerBackend, format)
	logging.SetBackend(loggerBackendFormatter)

	log.Info("info")
	log.Notice("notice")
	log.Warning("warning")
	log.Error("err")
	log.Critical("crit")

	// for {
	// test := `{"actuatorId": "asdf", "granularityValue":5, "granularityUnit":"MINUTE"}`
	// message := RcvMessage{"", -1, ""}
	// err := json.Unmarshal([]byte(test), &message)
	// if err != nil {
	// 	log.Info("Error unmarshalling the message:", err)
	// }
	// log.Info("actuatorId:", message.ActuatorId)
	// log.Info("granularityValue:", message.GranularityValue)
	// log.Info("granularityUnit:", message.GranularityUnit)

	// time.Sleep(1 * time.Second)
	// }

	err := godotenv.Load()
	if err != nil {
		log.Warning(err)
	}

	env := os.Getenv("ENV")
	log.Info("ENV:", string(env))

	rcvQueue := flag.String("q", "MoistureSensorScheduler", "The name of the queue")
	sendQueue := flag.String("s", "MoistureSensorTimerExpired", "The name of the queue")
	timeout := flag.Int64("t", 5, "How long, in seconds, that the message is hidden from others")
	messageHandle := flag.String("m", "", "The receipt handle of the message")
	flag.Parse()

	if *rcvQueue == "" {
		log.Info("You must supply a queue name for rcvQueue (-q QUEUE)")
		return
	}

	if *sendQueue == "" {
		log.Info("You must supply a queue name for sendQueue (-s QUEUE)")
		return
	}

	if *timeout < 0 {
		*timeout = 0
	}

	if *timeout > 12*60*60 {
		*timeout = 12 * 60 * 60
	}

	// set aws profile to aws-osuapp
	options := session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Config:            aws.Config{Region: aws.String("us-west-2")},
	}
	if env == "dev" {
		options.Profile = "aws-osuapp"
	}
	sess := session.Must(session.NewSessionWithOptions(options))

	// Create an SQS service client
	svc := sqs.New(sess)

	log.Info("made it here")

	// Get URL of queue
	rcvUrlResult, err := GetQueueURL(svc, rcvQueue)
	if err != nil {
		log.Error("Error getting the queue URL:", err)
		return
	}
	sendUrlResult, err := GetQueueURL(svc, sendQueue)
	if err != nil {
		log.Error("Error getting the queue URL:", err)
		return
	}

	rcvQueueURL := rcvUrlResult.QueueUrl
	sendQueueURL := sendUrlResult.QueueUrl

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

		log.Notice("Received", len(msgResult.Messages), "new message(s)")

		for _, msg := range msgResult.Messages {
			// log.Info("Message ID:     " + *msgResult.Messages[0].MessageId)
			// log.Info("Message Handle: " + *msgResult.Messages[0].ReceiptHandle)
			log.Info(*msg.Body)

			// save the body into a variable called rawMsg
			rawMsg := *msg.Body
			// convert to json
			message := RcvMessage{"", -1, ""}
			err = json.Unmarshal([]byte(rawMsg), &message)
			if err != nil {
				log.Error("Error unmarshalling rawMsg:", err)
				continue
			}

			err = handleMsg(svc, sendQueueURL, message)
			if err != nil {
				log.Error("Error handling message:", err)
				continue
			}

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

		// log.Info("Message deleted")
	}

	log.Critical("Program ended")
}

func handleMsg(svc *sqs.SQS, queueURL *string, message RcvMessage) error {
	actuatorId := message.ActuatorId
	granularityValue := message.GranularityValue
	granularityUnit := message.GranularityUnit

	if actuatorId == "" {
		return errors.New("actuatorId is nil")
	}
	if granularityValue < 0 {
		return errors.New("granularityValue is nil")
	}
	if granularityUnit == "" {
		return errors.New("granularityUnit is nil")
	}

	log.Info("actuatorId:", actuatorId)
	log.Info("granularityValue:", granularityValue)
	log.Info("granularityUnit:", granularityUnit)

	startTime := time.Now()
	endTime := startTime

	if granularityUnit == MINUTE {
		endTime = startTime.Add(time.Duration(granularityValue) * time.Minute)
	} else if granularityUnit == HOUR {
		endTime = startTime.Add(time.Duration(granularityValue) * time.Hour)
	} else if granularityUnit == DAY {
		endTime = startTime.AddDate(0, 0, granularityValue)
	} else {
		return errors.New("granularityUnit is not valid")
	}

	// temporarily set endTime to 10 seconds after startTime for testing
	endTime = startTime.Add(10 * time.Second)

	log.Info("startTime:", startTime)
	log.Info("endTime:", endTime)

	// start a timer that ends at endTime
	// when the timer ends, print a message and send a message to the sendQueue
	timer := time.NewTimer(endTime.Sub(startTime)) // endTime - startTime
	go func() {
		<-timer.C
		log.Info(actuatorId, "- Timer expired. Sending message to sendQueue.")
		message := SendMessage{actuatorId, false}
		messageBytes, err := json.Marshal(message)
		if err != nil {
			log.Error(actuatorId, " - Error marshalling message:", err)
		}
		strMsg := string(messageBytes)
		// send message to sendQueue
		sendErr := SendMsg(svc, queueURL, strMsg)
		if sendErr != nil {
			log.Error(actuatorId, " - Error sending message:", sendErr)
		}
	}()

	return nil
}

// GetQueueURL gets the URL of an Amazon SQS queue
// Inputs:
//
//	sess is the current session, which provides configuration for the SDK's service clients
//	queueName is the name of the queue
//
// Output:
//
//	If success, the URL of the queue and nil
//	Otherwise, an empty string and an error from the call to
func GetQueueURL(svc *sqs.SQS, queue *string) (*sqs.GetQueueUrlOutput, error) {
	result, err := svc.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: queue,
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// SendMsg sends a message to an Amazon SQS queue
// Inputs:
//
//	sess is the current session, which provides configuration for the SDK's service clients
//	queueURL is the URL of the queue
//
// Output:
//
//	If success, nil
//	Otherwise, an error from the call to SendMessage
func SendMsg(svc *sqs.SQS, queueURL *string, message string) error {
	_, err := svc.SendMessage(&sqs.SendMessageInput{
		DelaySeconds: aws.Int64(10),
		MessageBody:  aws.String(message),
		QueueUrl:     queueURL,
	})
	// snippet-end:[sqs.go.send_message.call]
	if err != nil {
		return err
	}

	return nil
}

// GetMessages gets the messages from an Amazon SQS queue
// Inputs:
//
//	sess is the current session, which provides configuration for the SDK's service clients
//	queueURL is the URL of the queue
//	timeout is how long, in seconds, the message is unavailable to other consumers
//
// Output:
//
//	If success, the latest message and nil
//	Otherwise, nil and an error from the call to ReceiveMessage
func GetMessages(svc *sqs.SQS, queueURL *string, timeout *int64) (*sqs.ReceiveMessageOutput, error) {
	// snippet-start:[sqs.go.receive_messages.call]
	msgResult, err := svc.ReceiveMessage(&sqs.ReceiveMessageInput{
		AttributeNames: []*string{
			aws.String(sqs.MessageSystemAttributeNameSentTimestamp),
		},
		MessageAttributeNames: []*string{
			aws.String(sqs.QueueAttributeNameAll),
		},
		QueueUrl:            queueURL,
		MaxNumberOfMessages: aws.Int64(10), // can process up to 10 messages at a time
		VisibilityTimeout:   timeout,
	})
	// snippet-end:[sqs.go.receive_messages.call]
	if err != nil {
		return nil, err
	}

	return msgResult, nil
}

// DeleteMessage deletes a message from an Amazon SQS queue
// Inputs:
//
//	sess is the current session, which provides configuration for the SDK's service clients
//	queueURL is the URL of the queue
//	messageID is the ID of the message
//
// Output:
//
//	If success, nil
//	Otherwise, an error from the call to DeleteMessage
func DeleteMessage(svc *sqs.SQS, queueURL *string, messageHandle *string) error {
	_, err := svc.DeleteMessage(&sqs.DeleteMessageInput{
		QueueUrl:      queueURL,
		ReceiptHandle: messageHandle,
	})
	// snippet-end:[sqs.go.delete_message.call]
	if err != nil {
		return err
	}

	return nil
}
