package pubsub

import (
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
	handler func(T),
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
		for delivery := range deliveryChan {
			var val T
			err := json.Unmarshal(delivery.Body, &val)
			if err != nil {
				fmt.Println("Error unmarshaling delivery body: ", err)
				continue
			}
			handler(val)
			err = delivery.Ack(false)
			if err != nil {
				fmt.Println("Error while acknowledging: ", err)
				continue
			}
		}
	}()
	return nil
}
