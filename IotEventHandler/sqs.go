package main

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
)

func sendSQS(sess *session.Session, sendQueue *string, messageBytes []byte) error {
	if *sendQueue == "" {
		fmt.Println("You must supply a queue name for sendQueue (-s QUEUE)")
		// return string error message
		return fmt.Errorf("You must supply a queue name for sendQueue (-s QUEUE)")
	}

	// Create an SQS service client
	svc := sqs.New(sess)

	sendUrlResult, err := GetQueueURL(svc, sendQueue)
	if err != nil {
		fmt.Errorf("Error getting the queue URL:", err)
		return err
	}

	queueURL := sendUrlResult.QueueUrl

	if err != nil {
		fmt.Errorf(*sendQueue, "- Error marshalling message:", err)
		return err
	}
	strMsg := string(messageBytes)
	// send message to sendQueue
	sendErr := SendMsg(svc, queueURL, strMsg)
	if sendErr != nil {
		fmt.Errorf(*sendQueue, "- Error sending message:", sendErr)
		return sendErr
	}

	return nil
}

const MINUTE = "MINUTES"
const HOUR = "HOURS"
const DAY = "DAYS"

// create enum for granularityUnit of minute, hour, day from json
type GranularityUnit string

const (
	Minute GranularityUnit = MINUTE
	Hour   GranularityUnit = HOUR
	Day    GranularityUnit = DAY
)

type SchedulerMessage struct {
	// name of the key must be capitalized to be exported
	ActuatorId       string          `json:"actuatorId"`
	GranularityValue int             `json:"granularityValue"`
	GranularityUnit  GranularityUnit `json:"granularityUnit"`
}

type ResetActuatorMessage struct {
	// name of the key must be capitalized to be exported
	ActuatorId string `json:"actuatorId"`
	Key        string `json:"key"`
	Value      string `json:"value`
}

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

func GetQueueURL(svc *sqs.SQS, queue *string) (*sqs.GetQueueUrlOutput, error) {
	result, err := svc.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: queue,
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}
