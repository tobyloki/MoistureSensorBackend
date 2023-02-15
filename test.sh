#!/bin/bash

# get args deviceId and onOff
deviceId=$1

# get current time
now=$(date +"%T")

# print hello
echo "$now, $deviceId - Bash starting timer for 5 seconds"

# wait for 5 seconds
sleep 5

# get current time
now=$(date +"%T")

# print out fake data
echo "$now, $deviceId - temperature: 20"
echo "$now, $deviceId - pressure: 100"
echo "$now, $deviceId - moisture: 50"


# get current time
now=$(date +"%T")

# print hello
echo "$now, $deviceId - Bash script done"