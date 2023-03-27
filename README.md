# gRPC with Go

Used for server-client communication with smart hub.

- Setup proto files (data structures) from within root project directory

```bash
protoc --go_out=. --go_opt=paths=source_relative \
 --go-grpc_out=. --go-grpc_opt=paths=source_relative \
 message/message.proto
```

- Run server

```bash
go run server/*.go
```

- Run client (must be run from client directory b/c it references a bash script inside of its relative directory)

```bash
cd client
go run .
```

# Docker

Local testing

```bash
sudo docker-compose up --build
```

Deploy to Docker Hub

```bash
sudo docker build -t grpc-server .

docker tag grpc-server:latest tobyloki/moisture-sensor-grpc-server:latest

docker push tobyloki/moisture-sensor-grpc-server:latest
```

EC2 Deployment

1. Launch EC2 instance with Linux AMI v2 arm64 on t4g.small instance
2. Add this under user data under advanced details (this install Docker and starts the service)

```bash
#! /bin/sh
yum update -y
amazon-linux-extras install docker
service docker start
usermod -a -G docker ec2-user
chkconfig docker on
```

3. Attach IAM role with permissions to access SQS and AWSLambdaBasicExecutionRole (for cloudwatch logs)
4. SSH into instance and pull the image

```bash
docker pull tobyloki/moisture-sensor-grpc-server:latest
```

5. Run the container. This command will also restart the container at startup even if the EC2 instance is restarted. It also forwards all logs to cloudwatch logs.

```bash
# Without aws cloudwatch logs
docker run -d --publish 50051:50051 --name grpc-serverContainer --restart unless-stopped tobyloki/moisture-sensor-grpc-server:latest

# With aws cloudwatch logs
docker run -d --publish 50051:50051 --name grpc-serverContainer --restart unless-stopped --log-driver=awslogs --log-opt awslogs-region=us-west-2 --log-opt awslogs-group=moisture-sensor-grpc-server --log-opt awslogs-create-group=true tobyloki/moisture-sensor-grpc-server:latest
```

6. Make sure the security group allows inbound traffic on port 50051

Extra stuff

```bash
docker rm -f grpc-serverContainer

docker logs --follow --since $(date +%Y-%m-%dT%H:%M:%SZ) grpc-serverContainer
```
