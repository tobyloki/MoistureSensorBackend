package main

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("hub-data-reporter")

func main() {
	// set up logging
	format := logging.MustStringFormatter(
		`%{color}%{time:15:04:05.000} %{shortfunc} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}`,
	)
	loggerBackend := logging.NewLogBackend(os.Stderr, "", 0)
	loggerBackendFormatter := logging.NewBackendFormatter(loggerBackend, format)
	logging.SetBackend(loggerBackendFormatter)

	log.Notice("Running...")

	// get node ids from chip-tool
	nodeIds, err := getNodeIds()
	if err != nil {
		log.Error("failed getNodeIds:", err)
	}

	// loop through all nodeIds and run task in parallel and wait for all tasks to finish
	waitGroup := sync.WaitGroup{}
	for _, nodeId := range nodeIds {
		waitGroup.Add(1)
		go func(nodeId string) {
			defer waitGroup.Done()
			// get data from chip-tool
			data, err := getData(nodeId)
			if err != nil {
				log.Error(nodeId, "- failed getData:", err)
			}

			// send data to the api
			err = sendData(*data)
			if err != nil {
				log.Error(nodeId, "- failed sendData:", err)
			}
		}(nodeId)
	}

	// wait for all tasks to finish
	waitGroup.Wait()
}

func getData(deviceId string) (*DataReport, error) {
	// run the bash command test.sh
	// Define the path to the bash script
	scriptPath := "./test.sh"

	// Create the command to run the script, passing in the args deviceId and onOff
	// NOTE: command line args only accept strings
	cmd := exec.Command(scriptPath, deviceId)

	// Run the command and capture its output in real time
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Errorf("Failed to run script: %s\n", err)
		return nil, err
	}

	// log.Infof("Script output: \n%s\n", output)

	data := DataReport{DeviceId: deviceId}

	// split the output in array by new line
	outputArray := strings.Split(string(output), "\n")
	for _, line := range outputArray {
		log.Info(line)

		if strings.Contains(line, "temperature: ") {
			tempStr := strings.Split(line, ": ")[1]
			// convert temperature to int
			temp, err := strconv.Atoi(tempStr)
			if err != nil {
				return nil, err
			}
			data.Temperature = temp
		} else if strings.Contains(line, "pressure: ") {
			pressureStr := strings.Split(line, ": ")[1]
			// convert pressure to int
			pressure, err := strconv.Atoi(pressureStr)
			if err != nil {
				return nil, err
			}
			data.Pressure = pressure
		} else if strings.Contains(line, "moisture: ") {
			moistureStr := strings.Split(line, ": ")[1]
			// convert moisture to int
			moisture, err := strconv.Atoi(moistureStr)
			if err != nil {
				return nil, err
			}
			data.Moisture = moisture
		}
	}

	return &data, nil
}

func getNodeIds() ([]string, error) {
	return []string{"32", "1"}, nil
}
