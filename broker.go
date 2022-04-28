// Package pubsub implements a topic-based publish-subscribe system using channels.
//
// Subscribers receive updates on channels provided to them when they subscribe to a topic. Topics
// are automatically created when you subscribe to them and they do not exist yet. In that case, a
// Publisher type is returned as well, providing methods to publish updates on the new topic. Topics
// are removed when a subscriber unsubscribes from it and there are no other subscribers left.
// Publishers include a stop channel from which reading only succeeds after the topic was removed.
package pubsub

import (
	"errors"
	"sync"
	"time"
)

// Broker handles subscribing and unsubcribing to topics.
type Broker[U any] struct {
	μ sync.Mutex

	publishers  map[string]<-chan U          // publishers' update channels by topic
	stops       map[string]chan<- struct{}   // channels the publishers listen on for a stop signal; indexed by topic
	subscribers map[string]map[chan<- U]bool // set of subscribers by topic

	notifyAboutTopic chan string // channel on which publishers notify the broker about news for a certain topic
}

// NewBroker creates a broker and starts a goroutine handling message distribution.
func NewBroker[U any]() *Broker[U] {
	b := &Broker[U]{
		publishers:       map[string]<-chan U{},
		stops:            map[string]chan<- struct{}{},
		subscribers:      map[string]map[chan<- U]bool{},
		notifyAboutTopic: make(chan string),
	}

	go b.loop()

	return b
}

// Subscribe returns a new channel on which to receive updates on a certain topic.
// Subscribe makes sure the topic exists by creating it if neccessary. When a new
// topic was created, a corresponding publisher is returned, otherwise newPublisher
// is nil.
func (b *Broker[U]) Subscribe(topic string) (updates chan U, newPublisher *Publisher[U]) {
	b.μ.Lock()
	defer b.μ.Unlock()

	newPublisher = b.createTopicIfNotExists(topic)
	updates = make(chan U)
	b.subscribers[topic][updates] = true

	return
}

// createTopicIfNotExists checks if a topic already exists. If so, nil is returned. If not, the topic
// and a new publisher are created, and the new publisher is returned.
func (b *Broker[U]) createTopicIfNotExists(topic string) *Publisher[U] {
	if _, ok := b.publishers[topic]; ok {
		return nil
	}

	publisher, updates, stop := newPublisher[U](topic, b.notifyAboutTopic)

	b.publishers[topic] = updates
	b.stops[topic] = stop
	b.subscribers[topic] = map[chan<- U]bool{}

	return publisher
}

// Unsubscribe removes the specified channel from the topic, meaning there will be no more messages sent to updates.
// Unsubscribe will close updates.
func (b *Broker[U]) Unsubscribe(updates chan U, topic string) error {
	b.μ.Lock()
	defer b.μ.Unlock()

	if _, ok := b.publishers[topic]; !ok {
		return errors.New("no such topic")
	}

	delete(b.subscribers[topic], updates)
	close(updates)
	b.removeTopicIfNoSubs(topic)

	return nil
}

// stopPublisherIfNoSubs checks if there are subscribers for the topic left. If not, it signals the topic's publisher
// to stop sending updates and removes the topic.
func (b *Broker[U]) removeTopicIfNoSubs(topic string) {
	if subscribers, ok := b.subscribers[topic]; !ok || len(subscribers) != 0 {
		return
	}

	close(b.stops[topic])
	b.removeTopic(topic)
}

// removeTopic closes all subscribers' update channels to signal that there will
// be no more updates on the topic, then removes the topic entirely. It assumes
// that the upstream publisher channel is already closed.
func (b *Broker[U]) removeTopic(topic string) {
	for subscriber := range b.subscribers[topic] {
		close(subscriber)
	}

	delete(b.subscribers, topic)
	delete(b.publishers, topic)
	delete(b.stops, topic)
}

// loop handles distribution of published updates as well as removing topics
// when the publisher responsible closes the update channel.
func (b *Broker[U]) loop() {
	for topic := range b.notifyAboutTopic {
		b.μ.Lock()

		// ignore publishers whose topic doesn't exist anymore
		updates, ok := b.publishers[topic]
		if !ok {
			b.μ.Unlock()
			continue
		}

		update, ok := <-updates
		if ok {
			for subscriber := range b.subscribers[topic] {
				select {
				case subscriber <- update:
				case <-time.After(10 * time.Millisecond):
					// 10ms timeout for each subscriber to receive
				}
			}
		} else {
			b.removeTopic(topic)
		}

		b.μ.Unlock()
	}
}
