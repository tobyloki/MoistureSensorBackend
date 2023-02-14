package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"encoding/json"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/joho/godotenv"
)

// lambda handler to receive events from sqs
func handler(ctx context.Context, sqsEvent events.SQSEvent) error {
	// create session
	err := godotenv.Load()
	if err != nil {
		fmt.Println(err)
	}
	env := os.Getenv("ENV")
	fmt.Println("ENV:", string(env))

	options := session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Config:            aws.Config{Region: aws.String("us-west-2")},
	}
	if env == "dev" {
		options.Profile = "aws-osuapp"
	}
	sess := session.Must(session.NewSessionWithOptions(options))

	// loop through messages (should only be one based on cloudformation configuration)
	for _, message := range sqsEvent.Records {
		fmt.Printf("The message %s for event source %s = %s \n", message.MessageId, message.EventSource, message.Body)

		// map message.Body to ResetActuatorMessage
		var resetActuatorMsg ResetActuatorMessage
		err := json.Unmarshal([]byte(message.Body), &resetActuatorMsg)
		if err != nil {
			fmt.Println("error:", err)
			return err
		}

		schedulerQueue := "MoistureSensorScheduler"
		updateActuatorQueue := "MoistureSensorUpdateActuator"

		// get actuator
		actuator, err := getActuator(resetActuatorMsg.ActuatorId)
		if err != nil {
			fmt.Println("error:", err)
			return err
		}
		fmt.Println("Actuator:", actuator)

		// check if sensor is in normal state from iot events
		isNormal, err := isNormalState(sess, resetActuatorMsg.SensorThingName)
		if err != nil {
			fmt.Println("error:", err)
			return err
		}

		if *isNormal {
			// send SQS to update actuator to on
			message := UpdateActuatorMessage{resetActuatorMsg.ActuatorId, "state", "on"}
			messageBytes, err := json.Marshal(message)
			if err != nil {
				fmt.Println("Error marshalling message:", err)
				return err
			}
			err = sendSQS(sess, &updateActuatorQueue, messageBytes)
			if err != nil {
				fmt.Println("Error sending SQS:", err)
				return err
			}

			fmt.Println(actuator.ActuatorId, "- Actuator updated to state on")
		} else {
			// update actuator expiration
			updateActuatorExpirationTimestamp(*actuator)

			// reset timer
			schedulerMsg := SchedulerMessage{resetActuatorMsg.SensorThingName, actuator.ActuatorId, actuator.GranularityValue, actuator.GranularityUnit}
			schedulerMsgBytes, err := json.Marshal(schedulerMsg)
			if err != nil {
				fmt.Println("Error marshalling message:", err)
				return err
			}
			err = sendSQS(sess, &schedulerQueue, schedulerMsgBytes)
			if err != nil {
				fmt.Println("Error sending SQS:", err)
				return err
			}

			fmt.Println(actuator.ActuatorId, "- Device is not in Normal state. Reset timer and updated expiration for actuator.")
		}
	}

	return nil
}

func main() {
	lambda.Start(handler)
}
