package pubsub

import (
	"context"
	"encoding/json"

	amqp "github.com/rabbitmq/amqp091-go"
)

type SimpleQueueType int

const (
	Transient SimpleQueueType = iota
	Durable
)

func DeclareAndBind(
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
) (*amqp.Channel, *amqp.Queue, error) {

	connCh, err := conn.Channel()
	if err != nil {
		return nil, &amqp.Queue{}, err
	}

	// declare the queue
	queue, err := connCh.QueueDeclare(
		queueName,
		queueType == Durable,
		queueType == Transient,
		queueType == Transient,
		false,
		nil,
	)

	// bind it to the exchange
	err = connCh.QueueBind(
		queueName,
		key,
		exchange,
		false,
		nil,
	)

	return connCh, &queue, err
}

func PublishJSON[T any](ch *amqp.Channel, exchange, key string, val T) error {
	bytes, err := json.Marshal(val)
	if err != nil {
		return err
	}
	err = ch.PublishWithContext(
		context.Background(),
		exchange,
		key,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        bytes,
		},
	)
	return err
}
