#!/bin/bash

# get args deviceId and onOff
deviceId=$1
onOff=$2

# get current time
now=$(date +"%T")

# print hello
echo "$now - Bash staring timer for 10 seconds: $deviceId, $onOff"

# wait for 10 seconds
sleep 10

# get current time
now=$(date +"%T")

# print hello
echo "$now - Bash script done: $deviceId, $onOff"