package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

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

	// run a loop every 10 seconds
	for {
		getDataAndSend()
		// note that this will wait only after getDataAndSend() has finished (which may take a while)
		time.Sleep(1 * time.Second)
	}
}

func getDataAndSend() {
	log.Notice("Running...")

	// get node ids from chip-tool
	nodeIds, err := getNodeIds()
	if err != nil {
		log.Error("failed getNodeIds:", err)
	}
	// print node ids
	log.Info("nodeIds:", nodeIds)

	// loop through all nodeIds and run task in parallel and wait for all tasks to finish
	// waitGroup := sync.WaitGroup{}

	// TODO: chip-tool doesn't like it when multiple commands are run at the same time b/c it tries to access the same resources:
	/*
		Session resumption cache deletion partially failed for fabric index 17, unable to delete node link: ../../src/controller/ExamplePersistentStorage.cpp:192: CHIP Error 0x000000AF: Write to file failed
	*/
	for _, nodeId := range nodeIds {
		// waitGroup.Add(1)
		// go func(nodeId string) {
		// 	defer waitGroup.Done()

		// get data from chip-tool
		data, err := getData(nodeId)
		if err != nil {
			log.Error(nodeId, "- failed getData:", err)
		} else {
			// send data to the api
			err = sendData(*data)
			if err != nil {
				log.Error(nodeId, "- failed sendData:", err)
			}
		}

		// }(nodeId)
	}

	// Wait for all goroutines to finish
	// waitGroup.Wait()
}

func getData(deviceId string) (*DataReport, error) {
	// run the bash command test.sh
	// Define the path to the bash script
	scriptPath := "./script.sh"

	/**********************/

	expiration := 15

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(expiration)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, scriptPath, deviceId)

	// Start the command
	// if err := cmd.Start(); err != nil {
	// 	log.Errorf("Failed to start command: %s\n", err)
	// 	return nil, err
	// }

	// Wait for the command to complete or the context to expire
	done := make(chan error, 1)
	output := make([]byte, 0)
	go func() {
		var err error
		output, err = cmd.CombinedOutput()
		done <- err
		close(done)
	}()

	select {
	case err := <-done:
		if err != nil {
			// delete temp-$deviceId.txt
			err = os.Remove("temp-" + deviceId + ".txt")
			if err != nil {
				// log.Error(err)
			}

			if ctx.Err() == context.DeadlineExceeded {
				// If the context has expired, try to kill the process
				if err := cmd.Process.Kill(); err != nil {
					log.Errorf("Failed to kill process: %s\n", err)
				}
				return nil, fmt.Errorf("Command timed out after %d seconds", expiration)
			}
			log.Errorf("Command failed: %s\n", err)
			return nil, err
		}
		// Process the output
		data := DataReport{DeviceId: deviceId}

		// split the output in array by new line
		outputArray := strings.Split(string(output), "\n")
		for _, line := range outputArray {
			log.Info(line)

			if strings.Contains(line, "temperature: ") {
				tempStr := strings.Split(line, ": ")[1]
				// convert to int
				temp, err := strconv.Atoi(tempStr)
				if err != nil {
					return nil, err
				}
				data.Temperature = temp
			} else if strings.Contains(line, "humidity: ") {
				humidityStr := strings.Split(line, ": ")[1]
				// convert to int
				humidity, err := strconv.Atoi(humidityStr)
				if err != nil {
					return nil, err
				}
				data.Humidity = humidity
			} else if strings.Contains(line, "pressure: ") {
				pressureStr := strings.Split(line, ": ")[1]
				// convert to int
				pressure, err := strconv.Atoi(pressureStr)
				if err != nil {
					return nil, err
				}
				data.Pressure = pressure
			} else if strings.Contains(line, "soilMoisture: ") {
				soilMoistureStr := strings.Split(line, ": ")[1]
				// convert to int
				soilMoisture, err := strconv.Atoi(soilMoistureStr)
				if err != nil {
					return nil, err
				}
				data.SoilMoisture = soilMoisture
			} else if strings.Contains(line, "light: ") {
				lightStr := strings.Split(line, ": ")[1]
				// convert to int
				light, err := strconv.Atoi(lightStr)
				if err != nil {
					return nil, err
				}
				data.Light = light
			}
		}

		return &data, nil
	case <-ctx.Done():
		// delete temp-$deviceId.txt
		err := os.Remove("temp-" + deviceId + ".txt")
		if err != nil {
			// log.Error(err)
		}

		// If the context has expired, try to kill the process
		if err = cmd.Process.Kill(); err != nil {
			log.Errorf("Failed to kill process: %s\n", err)
		}
		return nil, fmt.Errorf("Command timed out after %d seconds", expiration)
	}

	/**********************/
}

func getNodeIds() ([]string, error) {
	// return []string{"13"}, nil

	// read nodeIds.csv (each line is a nodeId)
	file, err := os.Open("./nodeIds.csv")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// read file line by line
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	var nodeIds []string
	for scanner.Scan() {
		// split text by comma
		textArray := strings.Split(scanner.Text(), ",")
		// nodeId is the first element
		nodeId := textArray[0]
		// device type is the second element
		deviceType := textArray[1]

		// if device type is "sensor"
		if deviceType == "sensor" {
			nodeIds = append(nodeIds, nodeId)
		}
	}

	return nodeIds, nil
}
