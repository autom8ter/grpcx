//go:generate mockgen -destination=./mocks/metrics.go -package=mocks . Metrics

package providers

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Metrics is an interface for collecting metrics
type Metrics interface {
	// Inc increments the value in a gauge
	Inc(name string, labels ...string)
	// Dec decrements the value in a gauge
	Dec(name string, labels ...string)
	// Observe records the value in a histogram
	Observe(name string, value float64, labels ...string)
	// Set sets the value in a gauge
	Set(name string, value float64, labels ...string)
	// RegisterHistogram registers a new histogram with the given name and labels
	RegisterHistogram(name string, labels ...string)
	// RegisterGauge registers a new gauge with the given name and labels
	RegisterGauge(name string, labels ...string)
}

// UnaryMetricsInterceptor returns a grpc unary server interceptor that collects metrics
func UnaryMetricsInterceptor(metrics Metrics) grpc.UnaryServerInterceptor {
	metrics.RegisterGauge("grpc_requests", "method", "code")
	metrics.RegisterHistogram("grpc_request_latency_seconds", "method")
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		now := time.Now()
		defer metrics.Observe("grpc_request_latency_ms", time.Since(now).Seconds(), info.FullMethod)
		resp, err := handler(ctx, req)
		if err != nil {
			code := status.Code(err)
			metrics.Inc("grpc_requests", info.FullMethod, code.String())
		} else {
			metrics.Inc("grpc_requests", info.FullMethod, codes.OK.String())
		}
		return resp, err
	}
}

// StreamMetricsInterceptor returns a grpc stream server interceptor that records metrics
func StreamMetricsInterceptor(metrics Metrics) grpc.StreamServerInterceptor {
	metrics.RegisterGauge("grpc_requests", "method", "code")
	metrics.RegisterHistogram("grpc_request_latency_seconds", "method")
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		now := time.Now()
		defer metrics.Observe("grpc_request_latency_seconds", time.Since(now).Seconds(), info.FullMethod)
		err := handler(srv, ss)
		if err != nil {
			metrics.Inc("grpc_requests", info.FullMethod)
		} else {
			metrics.Inc("grpc_requests", info.FullMethod)
		}
		return err
	}
}
