package pubsub

// Publisher provides methods to send updates to all subscribers of a certain topic.
type Publisher[U any] struct {
	// Stop will be closed by the broker when a subscriber unsubscribes and the topic is removed because there
	// are no subscribers left. This means you know to stop publishing updates when reading from Stop succeeds.
	// In that case, you should call Publisher.Close().
	Stop <-chan struct{}

	topic        string
	notifyPubSub chan<- string
	updates      chan<- U
}

func newPublisher[U any](topic string, notifyPubSub chan<- string) (*Publisher[U], <-chan U, chan<- struct{}) {
	updates := make(chan U, 1)
	stop := make(chan struct{})

	p := &Publisher[U]{
		topic:        topic,
		notifyPubSub: notifyPubSub,
		updates:      updates,
		Stop:         stop,
	}

	return p, updates, stop
}

// Topic returns the topic this publisher is meant to publish updates on.
func (p *Publisher[U]) Topic() string { return p.topic }

// Publish notifies p's broker that there is an update on p's topic and blocks until the broker received the notification.
// Publish then blocks until the broker received the update. Calling Publish() after Close() returns immediately. Use p's
// Stop channel to know when the broker stopped listening.
func (p *Publisher[U]) Publish(update U) {
	p.notifyPubSub <- p.topic
	select {
	case p.updates <- update:
		// will block when the broker stopped listening
	case <-p.Stop:
		// returns immediately when the broker stopped listening
	}
}

// Close tells the broker there will be no more updates coming from p. Calling Publish() after Close() returns immediately.
// Calling Close() makes the broker unsubscribe all subscribers and telling them updates on the topic have ended.
func (p *Publisher[U]) Close() {
	close(p.updates)
	p.notifyPubSub <- p.topic
}
