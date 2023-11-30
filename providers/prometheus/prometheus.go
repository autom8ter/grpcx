package prometheus

import (
	"context"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/spf13/viper"

	"github.com/autom8ter/grpcx/providers"
)

type Metrics struct {
	mu         sync.RWMutex
	gauges     map[string]*prometheus.GaugeVec
	histograms map[string]*prometheus.HistogramVec
}

// RegisterGauge registers a new gauge with the given name and labels
func (m *Metrics) RegisterGauge(name string, labels ...string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.gauges[name]; ok {
		return
	}
	m.gauges[name] = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: name,
	}, labels)
}

func (m *Metrics) RegisterHistogram(name string, labels ...string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.histograms[name]; ok {
		return
	}
	m.histograms[name] = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: name,
	}, labels)
}

func (m *Metrics) Inc(name string, labels ...string) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	gauge := m.gauges[name]
	gauge.WithLabelValues(labels...).Inc()
}

func (m *Metrics) Dec(name string, labels ...string) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	gauge := m.gauges[name]
	gauge.WithLabelValues(labels...).Dec()
}

func (m *Metrics) Observe(name string, value float64, labels ...string) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	hist := m.histograms[name]
	hist.WithLabelValues(labels...).Observe(value)
}

func (m *Metrics) Set(name string, value float64, labels ...string) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	gauge := m.gauges[name]
	gauge.WithLabelValues(labels...).Set(value)
}

func New() *Metrics {
	return &Metrics{
		gauges:     make(map[string]*prometheus.GaugeVec),
		histograms: make(map[string]*prometheus.HistogramVec),
	}
}

func Provider(ctx context.Context, cfg *viper.Viper) (providers.Metrics, error) {
	return New(), nil
}
