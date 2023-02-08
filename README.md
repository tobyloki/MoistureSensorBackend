# Moisture Sensor API

Build (`MoistureSensorApi.dll` located at `.aws-sam/build`)

```cmd
sam build --template-file cloudformation.yaml
```

Deploy cloudformation template to AWS from within root directory.

```cmd
sam deploy --config-file samconfig.toml --capabilities CAPABILITY_NAMED_IAM --no-confirm-changeset
```
