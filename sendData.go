package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type DataReport struct {
	// name of the key must be capitalized to be exported
	DeviceId    string `json:"deviceId"`
	Temperature int    `json:"temperature"`
	Pressure    int    `json:"pressure"`
	Moisture    int    `json:"moisture"`
}

func sendData(data DataReport) error {
	ret, err := httpRequest(data)
	if err != nil {
		return err
	}
	log.Info("ret:", *ret)

	// check if response contains error
	var root interface{}
	err = json.Unmarshal([]byte(*ret), &root)
	if err != nil {
		return err
	}
	rawJson := root.(map[string]interface{})
	error := rawJson["error"]
	if error != nil {
		return fmt.Errorf(*ret)
	}

	return nil
}

func httpRequest(data DataReport) (*string, error) {
	// send a get http request to a https://lcwdhzcciwo3d5amt623pobsxq0xuwwb.lambda-url.us-west-2.on.aws with parameters temperature, pressure, moisture
	url := "https://lcwdhzcciwo3d5amt623pobsxq0xuwwb.lambda-url.us-west-2.on.aws/report-data/" + data.DeviceId
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// add query parameters to the req
	q := req.URL.Query()
	q.Add("temperature", fmt.Sprint(data.Temperature))
	q.Add("pressure", fmt.Sprint(data.Pressure))
	q.Add("moisture", fmt.Sprint(data.Moisture))
	req.URL.RawQuery = q.Encode()

	log.Info("Sending to:", req.URL)
	// log.Info("req.URL.RawQuery:", req.URL.RawQuery)

	// send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	ret := string(body)

	// check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status code error: %s, response: %s", resp.Status, ret)
	}

	// return the response body
	return &ret, nil
}
