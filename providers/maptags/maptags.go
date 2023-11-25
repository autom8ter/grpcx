package maptags

import (
	"context"
	"sync"

	"github.com/spf13/viper"
	"google.golang.org/grpc/metadata"

	"github.com/autom8ter/grpcx/providers"
)

type MapTags struct {
	data    map[string]interface{}
	logTags []string
	mu      sync.RWMutex
}

func New(data map[string]interface{}, logTags []string) *MapTags {
	return &MapTags{
		data:    data,
		logTags: logTags,
	}
}

func (m *MapTags) LogTags() map[string]any {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(m.logTags) == 0 {
		return m.data
	}
	data := make(map[string]any)
	for _, tag := range m.logTags {
		if val, ok := m.data[tag]; ok {
			data[tag] = val
		}
	}
	return data
}

func (m *MapTags) WithMetadata(md metadata.MD) providers.Tags {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data["metadata"] = md
	return m
}

func (m *MapTags) WithMethod(method string) providers.Tags {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data["method"] = method
	return m
}

func (m *MapTags) WithContextID(contextID string) providers.Tags {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data["context_id"] = contextID
	return m
}

func (m *MapTags) WithError(err error) providers.Tags {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data["error"] = err
	return m
}

// Provider returns a ContextTagger that tags contexts with the given data and logTags
func Provider(ctx context.Context, cfg *viper.Viper) (providers.ContextTagger, error) {
	return providers.ContextTaggerFunc(func(ctx context.Context) providers.Tags {
		return New(map[string]interface{}{}, cfg.GetStringSlice("logging.tags"))
	}), nil
}

func (m *MapTags) GetMetadata() (metadata.MD, bool) {
	md, ok := m.data["metadata"].(metadata.MD)
	return md, ok
}

func (m *MapTags) GetMethod() (string, bool) {
	method, ok := m.data["method"].(string)
	return method, ok
}

func (m *MapTags) GetContextID() (string, bool) {
	contextID, ok := m.data["context_id"].(string)
	return contextID, ok
}

func (m *MapTags) GetError() (error, bool) {
	err, ok := m.data["error"].(error)
	return err, ok
}
