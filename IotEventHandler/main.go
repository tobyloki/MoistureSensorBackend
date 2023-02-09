package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
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

	deviceId := detector["keyValue"].(string)
	rawState := payload["state"].(map[string]interface{})
	state, err := json.Marshal(rawState)
	if err != nil {
		fmt.Println("Error marshalling state: ", err)
		return err
	}
	stateName := rawState["stateName"].(string)

	message := deviceId + " is now in " + stateName + " " + string(state)
	fmt.Println(message)

	// send SNS
	arn := "arn:aws:sns:us-west-2:978103014270:MoistureSensorPushNotificationTopic"
	ret, err := sendSNS(arn, message)
	if err != nil {
		return err
	}

	fmt.Println("SNS MessageId: ", *ret.MessageId)

	return nil
}

func sendSNS(arn string, message string) (*sns.PublishOutput, error) {
	options := session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Config:            aws.Config{Region: aws.String("us-west-2")},
	}
	sess := session.Must(session.NewSessionWithOptions(options))
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
