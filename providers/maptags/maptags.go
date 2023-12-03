package maptags

import (
	"sync"

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

func (m *MapTags) GetMetadata() (metadata.MD, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	md, ok := m.data["metadata"].(metadata.MD)
	return md, ok
}

func (m *MapTags) GetMethod() (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	method, ok := m.data["method"].(string)
	return method, ok
}

func (m *MapTags) GetContextID() (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	contextID, ok := m.data["context_id"].(string)
	return contextID, ok
}

func (m *MapTags) GetError() (error, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	err, ok := m.data["error"].(error)
	return err, ok
}

func (m *MapTags) Get(key string) (any, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	val, ok := m.data[key]
	return val, ok
}

func (m *MapTags) Set(key string, value any) providers.Tags {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = value
	return m
}

// NewTagsProvider returns a new TagsProvider that returns a new Tags instance
func NewTagsProvider(logTags []string) providers.TagsProvider {
	return func() providers.Tags {
		return &MapTags{
			data:    make(map[string]interface{}),
			logTags: logTags,
			mu:      sync.RWMutex{},
		}
	}
}
