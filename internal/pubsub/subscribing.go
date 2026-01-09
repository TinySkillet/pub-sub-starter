package pubsub

import (
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type AckType int

const (
	Ack AckType = iota
	NackRequeue
	NackDiscard
)

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
	handler func(T) AckType,
) error {
	// make sure queue exists and is bound to the exchange
	connCh, _, err := DeclareAndBind(
		conn,
		exchange,
		queueName,
		key,
		queueType,
	)

	if err != nil {
		return err
	}

	deliveryChan, err := connCh.Consume(
		queueName,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	go func() {
		var err error
		for delivery := range deliveryChan {
			var val T
			err = json.Unmarshal(delivery.Body, &val)
			if err != nil {
				fmt.Println("Error unmarshaling delivery body: ", err)
				continue
			}
			ackType := handler(val)
			switch ackType {
			case Ack:
				err = delivery.Ack(false)
			case NackRequeue:
				err = delivery.Nack(false, true)
			case NackDiscard:
				err = delivery.Nack(false, false)
			}
			if err != nil {
				fmt.Println("An error occured when n/acknowledging:", err)
				continue
			}
		}
	}()
	return nil
}
