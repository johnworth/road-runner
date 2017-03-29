package main

import (
	"github.com/cyverse-de/messaging"
	"github.com/streadway/amqp"
)

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
