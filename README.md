# Moisture Sensor Scheduler

Build (`main` executable located in `bin`)
Note: only x86_64 is supported at this time

```bash
GOOS=linux GOARCH=amd64 go build -o bin/main main.go
```

Deploy cloudformation template to AWS from within root directory.

```cmd
sam deploy --template-file cloudformation.yaml --capabilities CAPABILITY_NAMED_IAM --stack-name MoistureSensorScheduler --s3-bucket moisture-sensor-backend --s3-prefix cloudformation --profile aws-osuapp
```
