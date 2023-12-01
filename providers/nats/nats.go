package nats

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/spf13/viper"

	"github.com/autom8ter/grpcx/internal/utils"
	"github.com/autom8ter/grpcx/providers"
)

// Nats is a struct for a nats stream provider
type Nats struct {
	client *nats.Conn
}

// NewNats returns a new nats stream provider
func NewNats(client *nats.Conn) *Nats {
	return &Nats{client: client}
}

// Publish publishes a message to a topic
func (n *Nats) Publish(ctx context.Context, topic string, message map[string]any) error {
	message["_xid"] = uuid.NewString()
	message["_timestamp"] = time.Now().UnixMilli()
	bits, err := json.Marshal(message)
	if err != nil {
		return err
	}
	return n.client.Publish(topic, bits)
}

// Subscribe subscribes to a topic
func (n *Nats) Subscribe(ctx context.Context, topic, consumer string, handler providers.MessageHandler) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	sub, err := n.client.QueueSubscribeSync(topic, consumer)
	if err != nil {
		return utils.WrapError(err, "failed to subscribe")
	}
	defer sub.Unsubscribe()
	for {
		msg, err := sub.NextMsgWithContext(ctx)
		if err != nil {
			return err
		}
		var message = make(map[string]any)
		json.Unmarshal([]byte(msg.Data), &message)
		if !handler(ctx, message) {
			return nil
		}
	}
}

// AsyncSubscribe subscribes to a topic and calls the handler in a goroutine
func (n *Nats) AsyncSubscribe(ctx context.Context, topic, consumer string, handler providers.MessageHandler) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	sub, err := n.client.QueueSubscribeSync(topic, consumer)
	if err != nil {
		return utils.WrapError(err, "failed to subscribe")
	}
	defer sub.Unsubscribe()
	for {
		msg, err := sub.NextMsgWithContext(ctx)
		if err != nil {
			return err
		}
		var message = make(map[string]any)
		json.Unmarshal([]byte(msg.Data), &message)
		go func(msg *nats.Msg) {
			handlerCtx, handlerCancel := context.WithCancel(ctx)
			defer handlerCancel()
			if !handler(handlerCtx, message) {
				cancel()
			}
		}(msg)
	}
}

// Provider returns a new nats stream provider
func Provider(ctx context.Context, config *viper.Viper) (providers.Stream, error) {
	conn, err := nats.Connect(config.GetString("stream.nats.url"))
	if err != nil {
		return nil, utils.WrapError(err, "failed to connect to nats")
	}
	return &Nats{
		client: conn,
	}, nil
}
