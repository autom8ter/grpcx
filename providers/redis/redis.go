package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/cast"

	"github.com/autom8ter/grpcx/providers"
)

// Redis is a struct for a redis cache/stream provider
type Redis struct {
	client *redis.Client
}

// NewRedis returns a new redis cache/stream provider
func NewRedis(client *redis.Client) *Redis {
	return &Redis{client: client}
}

// NewInMem returns a new in-memory redis provider(used for testing)
func NewInMem() (*Redis, error) {
	mr, err := miniredis.Run()
	if err != nil {
		return nil, err
	}
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	return NewRedis(client), nil
}

// Get gets a key
func (r Redis) Get(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, key).Result()
}

// Set sets a key
func (r Redis) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	return r.client.Set(ctx, key, value, ttl).Err()
}

// Delete deletes a key
func (r Redis) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}

// Lock locks a key
func (r Redis) Lock(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	return r.client.SetNX(ctx, key, "lock", ttl).Result()
}

// Unlock unlocks a key
func (r Redis) Unlock(ctx context.Context, key string, ttl time.Duration) error {
	return r.client.Del(ctx, key).Err()
}

// Once runs a function once for a given key
func (r Redis) Once(ctx context.Context, key string, ttl time.Duration, fn func(ctx context.Context) error) (bool, error) {
	ok, err := r.Lock(ctx, key, ttl)
	if err != nil {
		return false, err
	}
	if ok {
		return true, fn(ctx)
	}
	return false, nil
}

// Publish publishes a message to a topic
func (r Redis) Publish(ctx context.Context, topic string, message map[string]any) error {
	message["_xid"] = uuid.NewString()
	message["_timestamp"] = time.Now().UnixMilli()
	bits, err := json.Marshal(message)
	if err != nil {
		return err
	}
	return r.client.Publish(ctx, topic, string(bits)).Err()
}

// Subscribe subscribes to a topic
func (r Redis) Subscribe(ctx context.Context, topic, consumer string, handler providers.MessageHandler) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	ch := r.client.Subscribe(ctx, topic).Channel()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-ch:
			if !ok {
				return fmt.Errorf("channel closed")
			}
			var message = make(map[string]any)
			json.Unmarshal([]byte(msg.Payload), &message)
			id := cast.ToString(message["_xid"])
			gotLock, err := r.client.SetNX(ctx, id+consumer, "lock", 15*time.Minute).Result()
			if err != nil {
				return err
			}
			if !gotLock {
				continue
			}
			if !handler(ctx, message) {
				cancel()
				return nil
			}
		}
	}
}

// AsyncSubscribe subscribes to a topic and calls the handler in a goroutine
func (r Redis) AsyncSubscribe(ctx context.Context, topic, consumer string, handler providers.MessageHandler) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	ch := r.client.Subscribe(ctx, topic).Channel()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-ch:
			if !ok {
				return fmt.Errorf("channel closed")
			}
			var message = make(map[string]any)
			json.Unmarshal([]byte(msg.Payload), &message)
			id := cast.ToString(message["_xid"])
			gotLock, err := r.client.SetNX(ctx, id+consumer, "lock", 15*time.Minute).Result()
			if err != nil {
				return err
			}
			if !gotLock {
				continue
			}
			go func(msg map[string]any) {
				handlerCtx, handlerCancel := context.WithCancel(ctx)
				defer handlerCancel()
				if !handler(handlerCtx, message) {
					cancel()
				}
			}(message)
		}
	}
}
