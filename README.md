# Moisture Sensor Scheduler

Local testing

```bash
sudo docker-compose up --build
```

Deploy to Docker Hub

```bash
sudo docker build -t scheduler .

docker tag scheduler:latest tobyloki/moisture-sensor-scheduler:latest

docker push tobyloki/moisture-sensor-scheduler:latest
```

EC2 Deployment

1. Launch EC2 instance with Linux AMI v2 arm64
2. Add this under user data under advanced details (this install Docker and starts the service)

```bash
#! /bin/sh
yum update -y
amazon-linux-extras install docker
service docker start
usermod -a -G docker ec2-user
chkconfig docker on
```

3. Attach IAM role with permissions to access SQS
4. SSH into instance and pull the image

```bash
docker pull tobyloki/moisture-sensor-scheduler:latest
```

5. Run the container. This command will also restart the container at startup even if the EC2 instance is restarted. It also forwards all logs to cloudwatch logs.

```bash
docker run -d --name schedulerContainer --restart unless-stopped --log-driver=awslogs --log-opt awslogs-region=us-west-2 --log-opt awslogs-group=moisture-sensor-scheduler --log-opt awslogs-create-group=true tobyloki/moisture-sensor-scheduler:latest
```

Extra stuff

```bash
docker rm -f schedulerContainer

docker logs --follow schedulerContainer
```
