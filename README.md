# Moisture Sensor Backend

Build API Backend (`MoistureSensorApi.dll` located at `MoistureSensorApi/bin/Release/.net6.0/linux-arm64/publish`)

```cmd
dotnet publish -c Release --self-contained false -r linux-arm64
```

Deploy cloudformation template to AWS from within root directory.

```cmd
sam deploy --template-file cloudformation.yaml --capabilities CAPABILITY_NAMED_IAM --stack-name MoistureSensor --s3-bucket moisture-sensor-backend --s3-prefix cloudformation --profile aws-osuapp
```