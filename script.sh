#!/bin/bash

# get args deviceId and onOff
deviceId=$1

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

# get on/off
# ~/matter/MoistureSensorFirmware/esp-matter/connectedhomeip/connectedhomeip/out/host/chip-tool onoff read on-off $deviceId 1 --commissioner-name 5 > $filename

# get temperature
# ~/matter/MoistureSensorFirmware/esp-matter/connectedhomeip/connectedhomeip/out/host/chip-tool temperaturemeasurement read measured-value $deviceId 1 --commissioner-name 5 > $filename

# # read $filename and get the line that contains "MeasuredValue: " and save the value after the last space
# temperature=$(cat $filename | grep "MeasuredValue: " | awk '{print $NF}')
# echo "$deviceId - temperature: $temperature"

# get humidity
# ~/matter/MoistureSensorFirmware/esp-matter/connectedhomeip/connectedhomeip/out/host/chip-tool relativehumiditymeasurement read measured-value $deviceId 2 --commissioner-name 5 > $filename

# # read $filename and get the line that contains "measured value: " and save the value after the last space
# humidity=$(cat $filename | grep "measured value: " | awk '{print $NF}')
# echo "$deviceId - humidity: $humidity"

# # get pressure
# ~/matter/MoistureSensorFirmware/esp-matter/connectedhomeip/connectedhomeip/out/host/chip-tool pressuremeasurement read measured-value $deviceId 3 --commissioner-name 5 > $filename

# # read $filename and get the line that contains "MeasuredValue: " and save the value after the last space
# pressure=$(cat $filename | grep "MeasuredValue: " | awk '{print $NF}')
# echo "$deviceId - pressure: $pressure"

# # get soil moisture
~/matter/MoistureSensorFirmware/esp-matter/connectedhomeip/connectedhomeip/out/host/chip-tool flowmeasurement read measured-value $deviceId 4 --commissioner-name 5 > $filename

# read $filename and get the line that contains "measured value: " and save the value after the last space
soilMoisture=$(cat $filename | grep "MeasuredValue: " | awk '{print $NF}')
echo "$deviceId - soilMoisture: $soilMoisture"

# # get light
# ~/matter/MoistureSensorFirmware/esp-matter/connectedhomeip/connectedhomeip/out/host/chip-tool illuminancemeasurement read measured-value $deviceId 5 --commissioner-name 5 > $filename

# # read $filename and get the line that contains "measured value: " and save the value after the last space
# light=$(cat $filename | grep "MeasuredValue: " | awk '{print $NF}')
# echo "$deviceId - light: $light"

# delete $filename
rm $filename

# get elapsed time since start
elapsedTime=$((($(date +%s%N) - $start)/1000000))

# print elapsed time
echo "Elapsed time: $elapsedTime ms"