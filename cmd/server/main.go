package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	fmt.Println("Starting Peril server...")

	conn_str := "amqp://guest:guest@localhost:5672/"
	conn, err := amqp.Dial(conn_str)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	fmt.Println("Connected to Peril server!")

	connCh, _, err := pubsub.DeclareAndBind(
		conn,
		routing.ExchangePerilTopic,
		"game_logs",
		"game_logs.*",
		pubsub.Durable,
	)
	if err != nil {
		log.Fatal(err)
	}

	gamelogic.PrintServerHelp()

repl:
	for {
		input := gamelogic.GetInput()
		if len(input) < 1 {
			continue
		}

		firstCmd := input[0]
		switch firstCmd {
		case "pause":
			fmt.Println("Sending a pause message...")
			err = pubsub.PublishJSON(
				connCh,
				routing.ExchangePerilDirect,
				routing.PauseKey,
				routing.PlayingState{
					IsPaused: true,
				},
			)
			if err != nil {
				fmt.Println("Error sending a pause message: ", err)
			}
		case "resume":
			fmt.Println("Sending a resume message...")
			err = pubsub.PublishJSON(
				connCh,
				routing.ExchangePerilDirect,
				routing.PauseKey,
				routing.PlayingState{
					IsPaused: false,
				},
			)
			if err != nil {
				fmt.Println("Error sending a resume message: ", err)
			}

		case "quit":
			fmt.Println("Exiting...")
			break repl

		default:
			fmt.Println("Invalid command")
		}
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	signal.Notify(sigCh, syscall.SIGTERM)

	<-sigCh

	fmt.Println("\nShutting down...")
}
