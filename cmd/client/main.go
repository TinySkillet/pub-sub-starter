package main

import (
	"fmt"
	"log"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func handlePause(gs *gamelogic.GameState) func(routing.PlayingState) pubsub.AckType {
	return func(ps routing.PlayingState) pubsub.AckType {
		defer fmt.Print("> ")
		gs.HandlePause(ps)
		return pubsub.Ack
	}
}

func handleWar(gs *gamelogic.GameState) func(gamelogic.RecognitionOfWar) pubsub.AckType {
	return func(rw gamelogic.RecognitionOfWar) pubsub.AckType {
		defer fmt.Print("> ")
		outcome, _, _ := gs.HandleWar(rw)
		switch outcome {

		case gamelogic.WarOutcomeNotInvolved:
			return pubsub.NackRequeue

		case gamelogic.WarOutcomeNoUnits:
			return pubsub.NackDiscard

		case gamelogic.WarOutcomeOpponentWon:
			return pubsub.Ack

		case gamelogic.WarOutcomeYouWon:
			return pubsub.Ack

		case gamelogic.WarOutcomeDraw:
			return pubsub.Ack

		default:
			fmt.Printf("Error: unknown war outcome: %v\n", outcome)
			return pubsub.NackDiscard
		}
	}
}

func handleMove(connCh *amqp.Channel, gs *gamelogic.GameState) func(gamelogic.ArmyMove) pubsub.AckType {
	return func(am gamelogic.ArmyMove) pubsub.AckType {
		defer fmt.Print("> ")
		moveOutcome := gs.HandleMove(am)
		switch moveOutcome {

		case gamelogic.MoveOutComeSafe:
			return pubsub.Ack

		case gamelogic.MoveOutcomeMakeWar:
			routingKey := fmt.Sprintf("%s.%s", routing.WarRecognitionsPrefix, gs.GetUsername())
			err := pubsub.PublishJSON(
				connCh,
				routing.ExchangePerilTopic,
				routingKey,
				gamelogic.RecognitionOfWar{
					Attacker: am.Player,
					Defender: gs.GetPlayerSnap(),
				},
			)
			if err != nil {
				return pubsub.NackDiscard
			}
			return pubsub.NackRequeue

		default:
			return pubsub.NackDiscard
		}
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

	connCh, err := conn.Channel()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Connected to Peril server!")

	username, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatal(err)
	}

	queueName := fmt.Sprintf("%s.%s", routing.PauseKey, username)
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
	err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilTopic,
		armyMoveQueue,
		armyMoveRoutingKey,
		pubsub.Transient,
		handleMove(connCh, gameState),
	)
	if err != nil {
		log.Fatal(err)
	}

	warRoutingKey := fmt.Sprintf("%s.%s", routing.WarRecognitionsPrefix, "*")
	warQueue := routing.WarRecognitionsPrefix
	err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilTopic,
		warQueue,
		warRoutingKey,
		pubsub.Durable,
		handleWar(gameState),
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
