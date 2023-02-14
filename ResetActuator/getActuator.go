package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

func getActuator(actuatorId string) (*Actuator, error) {
	// get actuator from graphql endpoint

	query := fmt.Sprintf(`query MyQuery { getActuator(id: \"%s\") { id expirationValue expirationGranularity currentExpirationTimestamp _version } }`, actuatorId)
	// fmt.Println("query:", query)
	ret, err := httpRequest(query)
	if err != nil {
		return nil, err
	}
	// fmt.Println("ret:", *ret)

	// parse json
	var root interface{}
	err = json.Unmarshal([]byte(*ret), &root)
	if err != nil {
		return nil, err
	}
	// fmt.Println("root:", root)
	rawJson := root.(map[string]interface{})
	getActuatorRaw := rawJson["data"].(map[string]interface{})["getActuator"]

	if getActuatorRaw == nil {
		return nil, fmt.Errorf("actuator not found")
	}

	getActuator := getActuatorRaw.(map[string]interface{})

	var actuator Actuator
	actuator.ActuatorId = getActuator["id"].(string)
	actuator.GranularityValue = int(getActuator["expirationValue"].(float64))
	actuator.GranularityUnit = GranularityUnit(getActuator["expirationGranularity"].(string))
	actuator.Version = int(getActuator["_version"].(float64))

	if getActuator["currentExpirationTimestamp"] != nil {
		expirationTimestamp := getActuator["currentExpirationTimestamp"].(string)
		actuator.ExpirationTimestamp = &expirationTimestamp
	}

	return &actuator, nil
}

func updateActuatorExpirationTimestamp(actuator Actuator) error {
	// set new expiration timestamp to today
	// get current datetime
	// add granularity value to current datetime
	// set new expiration timestamp to current datetime + granularity value

	// get current datetime
	currentDatetime := time.Now()

	// add granularity value to current datetime
	var newExpirationTimestamp time.Time
	switch actuator.GranularityUnit {
	case DAY:
		newExpirationTimestamp = currentDatetime.AddDate(0, 0, actuator.GranularityValue)
	case HOUR:
		newExpirationTimestamp = currentDatetime.Add(time.Hour * time.Duration(actuator.GranularityValue))
	case MINUTE:
		newExpirationTimestamp = currentDatetime.Add(time.Minute * time.Duration(actuator.GranularityValue))
	default:
		return fmt.Errorf("invalid granularity unit: %s", actuator.GranularityUnit)
	}

	// format like 2006-01-02 15:04:05.999999 -0700 MST
	newExpirationTimestamp = newExpirationTimestamp.UTC()
	newExpirationTimestampStr := newExpirationTimestamp.Format("2006-01-02 15:04:05.999999 -0700 MST")

	// set new expiration timestamp to current datetime + granularity value
	query := fmt.Sprintf(`mutation MyMutation { updateActuator(input: {id: \"%s\", _version: %d, currentExpirationTimestamp: \"%s\"}) { id } }`, actuator.ActuatorId, actuator.Version, newExpirationTimestampStr)
	// fmt.Println("query:", query)
	ret, err := httpRequest(query)
	if err != nil {
		return err
	}
	fmt.Println("ret:", *ret)

	return nil
}

type Actuator struct {
	// name of the key must be capitalized to be exported
	ActuatorId          string          `json:"id"`
	GranularityValue    int             `json:"expirationValue"`
	GranularityUnit     GranularityUnit `json:"expirationGranularity"`
	ExpirationTimestamp *string         `json:"currentExpirationTimestamp"`
	Version             int             `json:"_version"`
}

func httpRequest(query string) (*string, error) {
	// send an http request to a graphql endpoint with header x-api-key

	var jsonStr = fmt.Sprintf(`{"query": "%s"}`, query)
	fmt.Println("jsonStr:", jsonStr)
	req, err := http.NewRequest("POST", "https://7h6nr2h6n5amtaadd5db7gbu2i.appsync-api.us-west-2.amazonaws.com/graphql", bytes.NewBuffer([]byte(jsonStr)))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	// set the x-api-key header
	req.Header.Set("x-api-key", "da2-gnn7q3s2izhrnis7hypn3zt7ue")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// fmt.Println("response Status:", resp.Status)
	// fmt.Println("response Headers:", resp.Header)
	body, _ := ioutil.ReadAll(resp.Body)
	ret := string(body)
	// fmt.Println("response Body:", ret)

	return &ret, nil
}
