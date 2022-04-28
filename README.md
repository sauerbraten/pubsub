# pubsub

A Go package implementing a topic-based publish-subscribe system using channels.

## Usage

Get the package:

	$ go get github.com/sauerbraten/pubsub

Import the package:

```go
import (
	"github.com/sauerbraten/pubsub"
)
```

Subscribers receive updates on channels provided to them when they subscribe to a topic. Topics are automatically created when you subscribe to them and they do not exist yet. In that case, a Publisher type is returned as well, providing methods to publish updates on the new topic. Topics are removed when a subscriber unsubscribes from it and there are no other subscribers left. Publishers include a stop channel from which reading only succeeds after the topic was removed.

## Documentation

[Full documentation at godoc.org.](https://godoc.org/github.com/sauerbraten/pubsub)

## Example

See [example/main.go](./example/main.go).
