# Moisture Sensor API

This is the API that all IoT devices will report their sensor data to. They simply provide data in the following format:

```
https://<host>/report-data/<deviceId>?temperature=<temperature>&pressure=<pressure>&moisture=<moisture>
```

Build (`MoistureSensorApi.dll` located at `.aws-sam/build`)

```cmd
sam build --template-file cloudformation.yaml
```

Deploy cloudformation template to AWS from within root directory.

```cmd
sam deploy --config-file samconfig.toml --capabilities CAPABILITY_NAMED_IAM --no-confirm-changeset
```
