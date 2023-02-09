FROM golang:1.20-alpine3.17 AS build

# Set destination for COPY
WORKDIR /app

# Download Go modules
COPY go.mod .
COPY go.sum .
RUN go mod download

# Copy the source code. Note the slash at the end, as explained in
# https://docs.docker.com/engine/reference/builder/#copy
COPY *.go ./

# Build
RUN go build -o /main

## Deploy
FROM gcr.io/distroless/base-debian10

WORKDIR /

COPY --from=build /main /main

ENTRYPOINT ["/main"]