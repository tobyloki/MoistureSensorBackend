# Moisture Sensor Scheduler

Build (`main` executable located in `bin`)

Note: Only x86_64 is supported for go1.x runtime, but arm64 is supported for custom bootstrap provided.al2 which is what this project uses. Details about this method found here:

- https://docs.aws.amazon.com/lambda/latest/dg/golang-package.html#golang-package-al2
- https://github.com/aws-samples/sessions-with-aws-sam/tree/master/go-al2

This command will build the binaries and place them in `.aws-sam/build` directory. It'll also autogenerate `template.yaml` and `build.toml` which'll be referenced by the deploy command automatically.

```bash
sam build --template-file cloudformation.yaml
```

Deploy cloudformation template to AWS from within root directory.

```cmd
sam deploy --config-file samconfig.toml --capabilities CAPABILITY_NAMED_IAM --no-confirm-changeset
```
