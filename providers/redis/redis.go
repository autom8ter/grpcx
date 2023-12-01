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
	"github.com/spf13/viper"

	"github.com/autom8ter/grpcx/providers"
)

type Redis struct {
	client *redis.Client
}

// NewRedis returns a new redis cache/stream provider
func NewRedis(client *redis.Client) *Redis {
	return &Redis{client: client}
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

// Provider returns a new Redis cache provider(cache.addr, cache.password, cache.db)
func Provider(ctx context.Context, config *viper.Viper) (providers.Cache, error) {
	if config.GetString("cache.addr") == "" {
		return nil, fmt.Errorf("no redis addr found (cache.addr)")
	}
	client := redis.NewClient(&redis.Options{
		Addr:     config.GetString("cache.addr"),
		Password: config.GetString("cache.password"),
		DB:       config.GetInt("cache.db"),
	})
	return NewRedis(client), nil
}

// StreamProvider returns a new Redis stream provider(stream.addr, stream.password, stream.db)
func StreamProvider(ctx context.Context, config *viper.Viper) (providers.Stream, error) {
	var (
		addr     = config.GetString("stream.addr")
		password = config.GetString("stream.password")
		db       = config.GetInt("stream.db")
	)
	if addr == "" {
		addr = config.GetString("cache.addr")
	}
	if password == "" {
		password = config.GetString("cache.password")
	}
	if db == 0 {
		db = config.GetInt("cache.db")
	}
	if addr == "" || password == "" || db == 0 {
		return nil, fmt.Errorf("configuration missing for redis stream provider(stream.addr, stream.password, stream.db)")
	}
	client := redis.NewClient(&redis.Options{
		Addr:     config.GetString("cache.addr"),
		Password: config.GetString("cache.password"),
		DB:       config.GetInt("cache.db"),
	})
	return NewRedis(client), nil
}

// InMemProvider returns a new in-memory redis provider(used for testing)
func InMemProvider(ctx context.Context, config *viper.Viper) (providers.Cache, error) {
	mr, err := miniredis.Run()
	if err != nil {
		return nil, err
	}
	client := redis.NewClient(&redis.Options{
		Addr:     mr.Addr(),
		Password: config.GetString("cache.password"),
		DB:       config.GetInt("cache.password"),
	})
	return NewRedis(client), nil
}

// InMemProvider returns a new in-memory redis stream provider(used for testing)
func InMemStreamProvider(ctx context.Context, config *viper.Viper) (providers.Stream, error) {
	mr, err := miniredis.Run()
	if err != nil {
		return nil, err
	}
	client := redis.NewClient(&redis.Options{
		Addr:     mr.Addr(),
		Password: config.GetString("cache.password"),
		DB:       config.GetInt("cache.db"),
	})
	return NewRedis(client), nil
}
