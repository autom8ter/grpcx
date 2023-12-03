package prometheus

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics is a prometheus metrics provider
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

// RegisterHistogram registers a new histogram with the given name and labels
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

// Inc increments the gauge by 1
func (m *Metrics) Inc(name string, labels ...string) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	gauge := m.gauges[name]
	gauge.WithLabelValues(labels...).Inc()
}

// Dec decrements the gauge by 1
func (m *Metrics) Dec(name string, labels ...string) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	gauge := m.gauges[name]
	gauge.WithLabelValues(labels...).Dec()
}

// Observe adds a new observation to the histogram
func (m *Metrics) Observe(name string, value float64, labels ...string) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	hist := m.histograms[name]
	hist.WithLabelValues(labels...).Observe(value)
}

// Set sets the gauge to the given value
func (m *Metrics) Set(name string, value float64, labels ...string) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	gauge := m.gauges[name]
	gauge.WithLabelValues(labels...).Set(value)
}

// New returns a new Metrics instance
func New() *Metrics {
	return &Metrics{
		gauges:     make(map[string]*prometheus.GaugeVec),
		histograms: make(map[string]*prometheus.HistogramVec),
	}
}
