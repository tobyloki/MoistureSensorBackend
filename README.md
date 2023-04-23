# Moisture Sensor API

This is the API that all IoT devices will report their sensor data to. You can also fetch the last reported sensor data from this endpoint.

Swagger

```
https://<host>/swagger
```

To send data (from hub)

```
GET https://<host>/report-data/<deviceId>?temperature=<temperature>&humidity=<humidity>&soilMoisture=<soilMoisture>&pressure=<pressure>&light=<light>
```

To read data (from app)

```
GET https://<host>/fetch-data/<deviceId>
```

Build (`MoistureSensorApi.dll` located at `.aws-sam/build`)

```cmd
sam build --template-file cloudformation.yaml
```

Deploy cloudformation template to AWS from within root directory.

```cmd
sam deploy --config-file samconfig.toml --capabilities CAPABILITY_NAMED_IAM --no-confirm-changeset
```
