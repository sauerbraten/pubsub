package pubsub

// Publisher provides methods to send updates to all subscribers of a certain topic.
type Publisher struct {
	// Stop will be closed by the broker when a subscriber unsubscribes and the topic is removed because there
	// are no subscribers left. This means you know to stop publishing updates when reading from Stop succeeds.
	// In that case, you should call Publisher.Close().
	Stop <-chan struct{}

	topic        string
	notifyPubSub chan<- string
	updates      chan<- []byte
}

func newPublisher(topic string, notifyPubSub chan<- string) (*Publisher, <-chan []byte, chan<- struct{}) {
	updates := make(chan []byte, 1)
	stop := make(chan struct{})

	p := &Publisher{
		topic:        topic,
		notifyPubSub: notifyPubSub,
		updates:      updates,
		Stop:         stop,
	}

	return p, updates, stop
}

// Publish notifies p's broker that there is an update on p's topic and blocks until the broker received the notification.
// Publish then blocks until the broker received the update. Calling Publish() after Close() blocks indefinitely. Calling
// Publish after p.Stop was closed by the broker blocks indefinitely.
func (p *Publisher) Publish(update []byte) {
	p.notifyPubSub <- p.topic
	p.updates <- update
}

// Close tells the broker there will be no more updates coming from p. Calling Publish() after Close() blocks indefinitely.
func (p *Publisher) Close() {
	close(p.updates)
	p.notifyPubSub <- p.topic
	p.notifyPubSub = nil
}
