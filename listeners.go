package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/cyverse-de/logcabin"
	"github.com/cyverse-de/messaging"
	"github.com/streadway/amqp"
)

// RegisterTimeLimitDeltaListener sets a function that listens for TimeLimitDelta
// messages on the given client.
func RegisterTimeLimitDeltaListener(client *messaging.Client, timeTracker *TimeTracker, invID string) {
	client.AddDeletableConsumer(
		amqpExchangeName,
		amqpExchangeType,
		messaging.TimeLimitDeltaQueueName(invID),
		messaging.TimeLimitDeltaRequestKey(invID),
		func(d amqp.Delivery) {
			d.Ack(false)

			running(client, job, "Received delta request")

			deltaMsg := &messaging.TimeLimitDelta{}
			err := json.Unmarshal(d.Body, deltaMsg)
			if err != nil {
				running(client, job, fmt.Sprintf("Failed to unmarshal time limit delta: %s", err.Error()))
				return
			}

			newDuration, err := time.ParseDuration(deltaMsg.Delta)
			if err != nil {
				running(client, job, fmt.Sprintf("Failed to parse duration string from message: %s", err.Error()))
				return
			}

			err = timeTracker.ApplyDelta(newDuration)
			if err != nil {
				running(client, job, fmt.Sprintf("Failed to apply time limit delta: %s", err.Error()))
				return
			}

			running(client, job, fmt.Sprintf("Applied time delta of %s. New end date is %s", deltaMsg.Delta, timeTracker.EndDate.UTC().String()))
		})
}

// RegisterTimeLimitRequestListener sets a function that listens for
// TimeLimitRequest messages on the given client.
func RegisterTimeLimitRequestListener(client *messaging.Client, timeTracker *TimeTracker, invID string) {
	client.AddDeletableConsumer(
		amqpExchangeName,
		amqpExchangeType,
		messaging.TimeLimitRequestQueueName(invID),
		messaging.TimeLimitRequestKey(invID),
		func(d amqp.Delivery) {
			d.Ack(false)

			running(client, job, "Received time limit request")

			timeLeft := int64(timeTracker.EndDate.Sub(time.Now())) / int64(time.Millisecond)
			err := client.SendTimeLimitResponse(invID, timeLeft)
			if err != nil {
				running(client, job, fmt.Sprintf("Failed to send time limit response: %s", err.Error()))
				return
			}

			running(client, job, fmt.Sprintf("Sent message saying that time left is %dms", timeLeft))
		})
}

// RegisterTimeLimitResponseListener sets a function that handles messages that
// are sent on the jobs exchange with the key for time limit responses. This
// service doesn't need these messages, this is just here to force the queue
// to get cleaned up when road-runner exits.
func RegisterTimeLimitResponseListener(client *messaging.Client, invID string) {
	client.AddDeletableConsumer(
		amqpExchangeName,
		amqpExchangeType,
		messaging.TimeLimitResponsesQueueName(invID),
		messaging.TimeLimitResponsesKey(invID),
		func(d amqp.Delivery) {
			d.Ack(false)
			logcabin.Info.Print(string(d.Body))
		})
}

// RegisterStopRequestListener sets a function that responses to StopRequest
// messages.
func RegisterStopRequestListener(client *messaging.Client, exit chan messaging.StatusCode, invID string) {
	client.AddDeletableConsumer(
		amqpExchangeName,
		amqpExchangeType,
		messaging.StopQueueName(invID),
		messaging.StopRequestKey(invID),
		func(d amqp.Delivery) {
			d.Ack(false)
			running(client, job, "Received stop request")
			exit <- messaging.StatusKilled
		})
}
