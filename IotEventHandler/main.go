package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/joho/godotenv"
)

func main() {
	lambda.Start(handler)
}

// lambda handler to receive json data
func handler(ctx context.Context, event map[string]interface{}) error {
	strEvent, err := json.Marshal(event)
	if err != nil {
		fmt.Println("Error marshalling event: ", err)
		return err
	}

	fmt.Println(string(strEvent))

	// parse json for data
	payload := event["payload"].(map[string]interface{})
	detector := payload["detector"].(map[string]interface{})

	thingName := detector["keyValue"].(string)
	rawState := payload["state"].(map[string]interface{})
	state, err := json.Marshal(rawState)
	if err != nil {
		fmt.Println("Error marshalling state: ", err)
		return err
	}
	fmt.Println("State: ", string(state))
	stateName := rawState["stateName"].(string)

	// get sensor from thingName
	sensor, err := getSensor(thingName)
	if err != nil {
		fmt.Println("err:", err)
		return err
	}

	message := (*sensor).Name + " is now in " + stateName + " state"
	fmt.Println(message)

	// create session
	err = godotenv.Load()
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

	// set the queue names
	schedulerQueue := "MoistureSensorScheduler"
	resetActuatorQueue := "MoistureSensorUpdateActuator"

	// print out the queue names
	fmt.Println("Scheduler queue:", schedulerQueue)
	fmt.Println("ResetActuator queue:", resetActuatorQueue)

	// get actuators associated to sensor
	actuatorList, err := getActuators((*sensor).Id)
	if err != nil {
		fmt.Println("err:", err)
		return err
	}

	// run a for loop to update each actuator concurrently
	turningOff := stateName != "Normal"

	var wg sync.WaitGroup
	wg.Add(len(actuatorList))

	fmt.Println("Actuator list:", actuatorList)

	for _, actuator := range actuatorList {
		go func(actuator Actuator) {
			defer wg.Done()

			// check if actuator is expired
			// if actuator is turning off, update expiration and send SQS to Scheduler
			if turningOff {
				// update actuator expiration timestamp in db
				err := updateActuatorExpirationTimestamp(actuator)
				if err != nil {
					fmt.Println("err:", err)
				}

				// send SQS to scheduler
				message := SchedulerMessage{actuator.ActuatorId, actuator.GranularityValue, actuator.GranularityUnit}
				messageBytes, err := json.Marshal(message)
				if err != nil {
					fmt.Println("Error marshalling message:", err)
				} else {
					err = sendSQS(sess, &schedulerQueue, messageBytes)
					if err != nil {
						fmt.Println(schedulerQueue, "- Error sending SQS:", err)
					}
				}

				// also send SQS to ResetActuator
				message2 := ResetActuatorMessage{actuator.ActuatorId, "state", "off"}
				messageBytes2, err := json.Marshal(message2)
				if err != nil {
					fmt.Println("Error marshalling message:", err)
				} else {
					err = sendSQS(sess, &resetActuatorQueue, messageBytes2)
					if err != nil {
						fmt.Println(resetActuatorQueue, "- Error sending SQS:", err)
					}
				}

				fmt.Println(actuator.ActuatorId, "- Actuator updated to state off")
			} else {
				// if actuator turning on and not expired, send SQS to ResetActuator

				message := ResetActuatorMessage{actuator.ActuatorId, "state", "on"}
				messageBytes, err := json.Marshal(message)
				if err != nil {
					fmt.Println("Error marshalling message:", err)
				} else {
					sendSqsFlag := false

					lastExpiration := actuator.ExpirationTimestamp
					if lastExpiration != nil {
						hasExpiredCheck, err := hasExpired(*lastExpiration)
						if err != nil {
							fmt.Println("Error checking expiration:", err)
						} else {
							if *hasExpiredCheck {
								sendSqsFlag = true
							}
						}
					} else {
						sendSqsFlag = true
					}

					if sendSqsFlag {
						// send SQS
						err = sendSQS(sess, &resetActuatorQueue, messageBytes)
						if err != nil {
							fmt.Println(resetActuatorQueue, "- Error sending SQS:", err)
						}

						fmt.Println(actuator.ActuatorId, "- Actuator updated to state on")
					} else {
						fmt.Println(actuator.ActuatorId, "- Actuator not updated due to expiration")
					}
				}
			}
		}(actuator)
	}

	wg.Wait()

	// send SNS
	arn := "arn:aws:sns:us-west-2:978103014270:MoistureSensorPushNotificationTopic"
	ret, err := sendSNS(sess, arn, message)
	if err != nil {
		return err
	}

	fmt.Println("SNS MessageId: ", *ret.MessageId)

	return nil
}

func hasExpired(timestamp string) (*bool, error) {
	sampleLayout := "2006-01-02 15:04:05.999999 -0700 MST"
	parsedTime, err := time.Parse(sampleLayout, timestamp)
	if err != nil {
		// fmt.Println("Error parsing timestamp:", err)
		return nil, err
	}

	if parsedTime.Before(time.Now()) {
		// fmt.Println("Timestamp has expired")
		return aws.Bool(true), nil
	} else {
		// fmt.Println("Timestamp has not expired")
		return aws.Bool(false), nil
	}
}

func sendSNS(sess *session.Session, arn string, message string) (*sns.PublishOutput, error) {
	svc := sns.New(sess)

	params := &sns.PublishInput{
		Message:  aws.String(message),
		TopicArn: aws.String(arn),
	}

	ret, err := svc.Publish(params)
	if err != nil {
		fmt.Println("Error sending SNS: ", err)
		return nil, err
	}

	return ret, nil
}
