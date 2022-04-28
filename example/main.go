package main

import (
	"fmt"
	"time"

	"github.com/sauerbraten/pubsub"
)

type Update struct {
	Seq int
	Msg string
}

func main() {
	broker := pubsub.NewBroker[Update]()

	// subscribe to topic
	updates, newPublisher := broker.Subscribe("topic")
	if newPublisher != nil {
		// maybe start producing updates to receive
		go publish(newPublisher)
	}

	// subscribe many goroutines to the same topic
	// and/or
	// subscribe one goroutine to many different topics (requires additional publishing goroutines)

	// receive updates until broker unsubscribes you
	for update := range updates {
		// process update
		fmt.Printf("received update %d: %s\n", update.Seq, update.Msg)
	}

	// or, at some point, just unsubscribe
	broker.Unsubscribe(updates, "topic")
}

func publish(pub *pubsub.Publisher[Update]) {
	seq := 0
	for {
		select {
		case <-pub.Stop:
			pub.Close()
			return

		case <-time.After(1 * time.Second):
			pub.Publish(Update{Seq: seq, Msg: time.Now().Format(time.RFC3339)})
			seq++
		}
	}
}
