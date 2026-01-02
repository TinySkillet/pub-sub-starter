package main

import (
	"fmt"
	"log"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func handlePause(gs *gamelogic.GameState) func(routing.PlayingState) {
	return func(ps routing.PlayingState) {
		defer fmt.Print("> ")
		gs.HandlePause(ps)
	}
}

func handleMove(gs *gamelogic.GameState) func(gamelogic.ArmyMove) {
	return func(am gamelogic.ArmyMove) {
		defer fmt.Print("> ")
		gs.HandleMove(am)
	}
}

func main() {
	fmt.Println("Starting Peril client...")

	conn_str := "amqp://guest:guest@localhost:5672/"
	conn, err := amqp.Dial(conn_str)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	fmt.Println("Connected to Peril server!")

	username, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatal(err)
	}

	queueName := fmt.Sprintf("%s.%s", routing.PauseKey, username)
	_, _, err = pubsub.DeclareAndBind(
		conn,
		routing.ExchangePerilDirect,
		queueName,
		routing.PauseKey,
		pubsub.Transient,
	)
	if err != nil {
		log.Fatal(err)
	}

	gameState := gamelogic.NewGameState(username)

	err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilDirect,
		queueName,
		routing.PauseKey,
		pubsub.Transient,
		handlePause(gameState),
	)
	if err != nil {
		log.Fatal(err)
	}

	armyMoveRoutingKey := fmt.Sprintf("%s.%s", routing.ArmyMovesPrefix, "*")
	armyMoveQueue := fmt.Sprintf("%s.%s", routing.ArmyMovesPrefix, username)

	connCh, _, err := pubsub.DeclareAndBind(
		conn,
		routing.ExchangePerilTopic,
		armyMoveQueue,
		armyMoveRoutingKey,
		pubsub.Transient,
	)
	if err != nil {
		log.Fatal(err)
	}

	err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilTopic,
		armyMoveQueue,
		armyMoveRoutingKey,
		pubsub.Transient,
		handleMove(gameState),
	)
	if err != nil {
		log.Fatal(err)
	}

repl:
	for {
		input := gamelogic.GetInput()
		if len(input) < 1 {
			continue
		}

		firstCommand := input[0]
		switch firstCommand {
		case "spawn":
			err := gameState.CommandSpawn(input)
			if err != nil {
				fmt.Println("Error spawning unit:", err)
			}

		case "move":
			armyMove, err := gameState.CommandMove(input)
			if err != nil {
				fmt.Println("Error moving unit: ", err)
				continue
			}
			err = pubsub.PublishJSON(
				connCh,
				routing.ExchangePerilTopic,
				armyMoveRoutingKey,
				armyMove,
			)
			if err != nil {
				fmt.Println("Error publishing move: ", err)
			}

		case "status":
			gameState.CommandStatus()

		case "help":
			gamelogic.PrintClientHelp()

		case "spam":
			fmt.Println("Spamming is not allowed yet!")

		case "quit":
			gamelogic.PrintQuit()
			break repl

		default:
			fmt.Println("Invalid command!")
			continue
		}
	}

}
