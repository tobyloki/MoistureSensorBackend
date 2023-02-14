package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ioteventsdata"
)

func isNormalState(sess *session.Session, detectorId string) (*bool, error) {
	// get iot events stuff
	iotEvents := ioteventsdata.New(sess)

	input := &ioteventsdata.DescribeDetectorInput{
		DetectorModelName: aws.String("MoistureSensorModel"),
		KeyValue:          aws.String(detectorId),
	}

	result, err := iotEvents.DescribeDetector(input)
	if err != nil {
		return nil, err
	}

	var stateName string = *(result.Detector.State.StateName)
	fmt.Println(detectorId, "is in", stateName, "state")

	if stateName == "Normal" {
		return aws.Bool(true), nil
	} else {
		return aws.Bool(false), nil
	}
}
