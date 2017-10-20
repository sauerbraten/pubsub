# pubsub

A Go package implementing a topic-based publish-subscribe system using channels.

## Usage

Get the package:

	$ go get github.com/sauerbraten/pubsub

Import the package:

	import (
		"github.com/sauerbraten/pubsub"
	)

Subscribers receive updates on channels provided to them when they subscribe to a topic. Topics are automatically created when you subscribe to them and they do not exist yet. In that case, a Publisher type is returned as well, providing methods to publish updates on the new topic. Topics are removed when a subscriber unsubscribes from it and there are no other subscribers left. Publishers include a stop channel from which reading only succeeds after the topic was removed.

## Documentation

[Full documentation at godoc.org.](https://godoc.org/github.com/sauerbraten/pubsub)

## Example

	package main

	import (
		"time"

		"github.com/sauerbraten/pubsub"
	)

	func main() {
		broker := pubsub.NewBroker()

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
		}

		// or, at some point, just unsubscribe
		broker.Unsubscribe(updates, "topic")
	}

	func publish(pub *pubsub.Publisher) {
		for {
			select {
			case <-pub.Stop:
				pub.Close()
				return

			case <-time.After(1 * time.Second):
				pub.Publish([]byte("update"))
			}
		}
	}
