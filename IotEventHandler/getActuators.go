package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

type Sensor struct {
	Id        string `json:"id"`
	Name      string `json:"name"`
	ThingName string `json:"thingName"`
}

func getSensor(thingName string) (*Sensor, error) {
	query := "query MyQuery { listSensors { items { id name thingName _deleted } } }"
	ret, err := httpRequest(query)
	if err != nil {
		return nil, err
	}
	fmt.Println("ret:", *ret)

	// parse out the sensor id from ret
	var root interface{}
	err = json.Unmarshal([]byte(*ret), &root)
	if err != nil {
		return nil, err
	}
	rawJson := root.(map[string]interface{})
	items := rawJson["data"].(map[string]interface{})["listSensors"].(map[string]interface{})["items"].([]interface{})

	for _, item := range items {
		if item.(map[string]interface{})["thingName"] == thingName {
			itemValue := item.(map[string]interface{})

			// check if deleted is not null and true
			if itemValue["_deleted"] != nil && itemValue["_deleted"].(bool) {
				continue
			}

			id := itemValue["id"].(string)
			name := itemValue["name"].(string)
			thingName := itemValue["thingName"].(string)

			sensor := Sensor{
				Id:        id,
				Name:      name,
				ThingName: thingName,
			}

			return &sensor, nil
		}
	}

	return nil, fmt.Errorf("sensor not found")
}

func getActuators(sensorId string) ([]Actuator, error) {
	integrationId, err := getIntegrationId(sensorId)
	if err != nil {
		// fmt.Println("err:", err)
		return nil, err
	}
	fmt.Println("integration id:", *integrationId)

	actuatorList, err := getActuatorList(*integrationId)
	if err != nil {
		// fmt.Println("err:", err)
		return nil, err
	}
	// fmt.Println("actuator list:", actuatorList)

	return actuatorList, nil
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

func getActuatorList(integrationId string) ([]Actuator, error) {
	// get actuator list
	query := fmt.Sprintf(`query MyQuery { getIntegration(id: \"%s\") { Actuators { items { id expirationValue expirationGranularity currentExpirationTimestamp _version _deleted } } } }`, integrationId)
	ret, err := httpRequest(query)
	if err != nil {
		return nil, err
	}
	fmt.Println("ret:", *ret)

	// parse out the actuator list from ret
	var root interface{}
	err = json.Unmarshal([]byte(*ret), &root)
	if err != nil {
		return nil, err
	}
	rawJson := root.(map[string]interface{})
	data := rawJson["data"].(map[string]interface{})
	getIntegration := data["getIntegration"]
	if getIntegration == nil {
		return nil, fmt.Errorf("getIntegration is nil")
	}
	actuators := getIntegration.(map[string]interface{})["Actuators"].(map[string]interface{})
	items := actuators["items"].([]interface{})
	actuatorList := make([]Actuator, len(items))
	for i, item := range items {
		itemJson := item.(map[string]interface{})

		// check if deleted is not null and true
		if itemJson["_deleted"] != nil && itemJson["_deleted"].(bool) {
			continue
		}

		// fmt.Println("item:", itemJson["id"])
		actuatorId := itemJson["id"].(string)
		granularityValue := int(itemJson["expirationValue"].(float64))
		granularityUnit := GranularityUnit(itemJson["expirationGranularity"].(string))
		expirationTimestamp := itemJson["currentExpirationTimestamp"]
		_version := itemJson["_version"].(float64)

		// declare variable of type *string

		// check expirationTimestamp for nil and set expiration variable
		var expiration *string
		if expirationTimestamp == nil {
			expiration = nil
		} else {
			expiration = new(string)
			*expiration = expirationTimestamp.(string)
		}

		actuatorList[i] = Actuator{
			ActuatorId:          actuatorId,
			GranularityValue:    granularityValue,
			GranularityUnit:     granularityUnit,
			ExpirationTimestamp: expiration,
			Version:             int(_version),
		}
	}
	return actuatorList, nil
}

func getIntegrationId(sensorId string) (*string, error) {
	// get integration id
	query := fmt.Sprintf(`query MyQuery { getSensor(id: \"%s\") { integrationID } }`, sensorId)
	ret, err := httpRequest(query)
	if err != nil {
		return nil, err
	}
	fmt.Println("ret:", *ret)

	// parse out the integration id from ret
	var root interface{}
	err = json.Unmarshal([]byte(*ret), &root)
	if err != nil {
		return nil, err
	}
	rawJson := root.(map[string]interface{})
	data := rawJson["data"].(map[string]interface{})
	getSensor := data["getSensor"]
	// check if getSensor is nil
	if getSensor == nil {
		return nil, fmt.Errorf("getSensor is nil")
	}
	integrationID := getSensor.(map[string]interface{})["integrationID"].(string)
	// check if integrationID is nil
	// fmt.Println("integrationID:", integrationID)
	return &integrationID, nil
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
