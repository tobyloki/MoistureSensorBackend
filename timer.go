package main

import (
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/service/sqs"
)

var (
	timers = make(map[string]*time.Timer)
	mu     sync.Mutex
)

func StartTimer(duration time.Duration, callback func(actuatorId string, svc *sqs.SQS, queueURL *string), actuatorId string, svc *sqs.SQS, queueURL *string) {
	mu.Lock()
	defer mu.Unlock()

	// If a timer already exists with the same actuatorID, stop it
	if t, ok := timers[actuatorId]; ok {
		t.Stop()
		log.Notice(actuatorId, "- Previous timer stopped")
	}

	// Create a new timer with the specified duration
	timers[actuatorId] = time.AfterFunc(duration, func() {
		mu.Lock()
		defer mu.Unlock()
		delete(timers, actuatorId)
		callback(actuatorId, svc, queueURL)
	})

	log.Info(actuatorId, "- Started timer for", duration)
}
