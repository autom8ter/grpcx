//go:generate mockgen -destination=./mocks/queue.go -package=mocks . Stream

package providers

import (
	"context"

	"github.com/spf13/viper"
)

// MessageHandler is a function that handles a message from a stream
type MessageHandler func(ctx context.Context, message map[string]any) bool

// Stream is an interface for a stream
type Stream interface {
	// Publish publishes a message to the stream
	Publish(ctx context.Context, topic string, message map[string]any) error
	// Subscribe subscribes to a topic
	Subscribe(ctx context.Context, topic string, consumer string, handler MessageHandler) error
	// AsyncSubscribe subscribes to a topic and handles messages asynchronously
	AsyncSubscribe(ctx context.Context, topic string, consumer string, handler MessageHandler) error
}

// StreamProvider is a function that returns a Stream
type StreamProvider func(ctx context.Context, cfg *viper.Viper) (Stream, error)
