#!/bin/bash

# get args deviceId and onOff
deviceId=$1
state=$2

# set filename to temp- + deviceId + .txt
filename="temp-$deviceId.txt"

# get start time to be able to calculate elapsed time
start=$(date +%s%N)

# print hello
echo "$deviceId - Bash script started"

# get current time
now=$(date +"%T")

# delete $filename only if it exists
if [ -f "$filename" ]; then
    rm $filename
fi

# if state is 1, run onoff on, else run onoff off
if [ $state -eq 1 ]; then
    ~/matter/MoistureSensorFirmware/esp-matter/connectedhomeip/connectedhomeip/out/host/chip-tool onoff on $deviceId 1 --commissioner-name 5 > $filename
else
    ~/matter/MoistureSensorFirmware/esp-matter/connectedhomeip/connectedhomeip/out/host/chip-tool onoff off $deviceId 1 --commissioner-name 5 > $filename
fi

# read $filename and get the line that contains "Status=" and save the value after it
status=$(grep "Status=" "$filename" | awk -F 'Status=' '{print $2}')
echo "$deviceId - status: $status"

# delete $filename
rm $filename

# get elapsed time since start
elapsedTime=$((($(date +%s%N) - $start)/1000000))

# print elapsed time
echo "Elapsed time: $elapsedTime ms"