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
type Broker struct {
	μ sync.Mutex

	publishers  map[string]<-chan []byte          // publishers' update channels by topic
	stops       map[string]chan<- struct{}        // channels the publishers listen on for a stop signal; indexed by topic
	subscribers map[string]map[chan<- []byte]bool // set of subscribers by topic

	notifyAboutTopic chan string // channel on which publishers notify the broker about news for a certain topic
}

// NewBroker creates a broker and starts a goroutine handling message distribution.
func NewBroker() *Broker {
	b := &Broker{
		publishers:       map[string]<-chan []byte{},
		stops:            map[string]chan<- struct{}{},
		subscribers:      map[string]map[chan<- []byte]bool{},
		notifyAboutTopic: make(chan string),
	}

	go b.loop()

	return b
}

// Subscribe returns a new channel on which to receive updates on a certain topic.
// Subscribe makes sure the topic exists by creating it if neccessary. When a new
// topic was created, a corresponding publisher is returned, otherwise newPublisher
// is nil.
func (b *Broker) Subscribe(topic string) (updates chan []byte, newPublisher *Publisher) {
	b.μ.Lock()
	defer b.μ.Unlock()

	newPublisher = b.createTopicIfNotExists(topic)
	updates = make(chan []byte)
	b.subscribers[topic][updates] = true

	return
}

// createTopicIfNotExists checks if a topic already exists. If so, nil is returned. If not, the topic
// and a new publisher are created, and the new publisher is returned.
func (p *Broker) createTopicIfNotExists(topic string) *Publisher {
	if _, ok := p.publishers[topic]; ok {
		return nil
	}

	publisher, updates, stop := newPublisher(topic, p.notifyAboutTopic)

	p.publishers[topic] = updates
	p.stops[topic] = stop
	p.subscribers[topic] = map[chan<- []byte]bool{}

	return publisher
}

// Unsubscribe removes the specified channel from the topic, meaning there will be no more messages sent to updates.
// Unsubscribe will close updates.
func (b *Broker) Unsubscribe(updates chan []byte, topic string) error {
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
func (p *Broker) removeTopicIfNoSubs(topic string) {
	if subscribers, ok := p.subscribers[topic]; !ok || len(subscribers) != 0 {
		return
	}

	close(p.stops[topic])
	p.removeTopic(topic)
}

// loop handles distribution of published updates as well as removing topics
// when the publisher responsible closes the update channel.
func (b *Broker) loop() {
	for {
		select {
		case topic := <-b.notifyAboutTopic:
			b.μ.Lock()

			updates, ok := b.publishers[topic]
			if !ok {
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
}

// removeTopic closes all subscribers' update channels to signal that there will
// be no more updates on the topic, then removes the topic entirely. It assumes
// that the upstream publisher channel is already closed.
func (p *Broker) removeTopic(topic string) {
	for subscriber := range p.subscribers[topic] {
		close(subscriber)
	}

	delete(p.subscribers, topic)
	delete(p.publishers, topic)
	delete(p.stops, topic)
}
