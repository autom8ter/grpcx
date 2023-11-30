package grpcx

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/autom8ter/protoc-gen-authorize/authorizer"
	"github.com/autom8ter/protoc-gen-ratelimit/limiter"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/auth"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	grpc_validator "github.com/grpc-ecosystem/go-grpc-middleware/validator"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/palantir/stacktrace"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/cors"
	"github.com/soheilhy/cmux"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/autom8ter/grpcx/internal/utils"
	"github.com/autom8ter/grpcx/providers"
	"github.com/autom8ter/grpcx/providers/maptags"
	slog2 "github.com/autom8ter/grpcx/providers/slog"
)

type serverOpt struct {
	GatewayOpts        []runtime.ServeMuxOption
	UnaryInterceptors  []grpc.UnaryServerInterceptor
	StreamInterceptors []grpc.StreamServerInterceptor
	Logger             providers.LoggingProvider
	Database           providers.DatabaseProvider
	Cache              providers.CacheProvider
	Stream             providers.StreamProvider
	Auth               grpc_auth.AuthFunc
	Authz              []authorizer.Authorizer
	Tagger             providers.ContextTaggerProvider
	RateLimit          limiter.Limiter
	Metrics            providers.MetricsProvider
	Handlers           []CustomHTTPRoute
	GrpcHealthCheck    grpc_health_v1.HealthServer
}

// ServerOption is a function that configures the server. All ServerOptions are optional.
type ServerOption func(opt *serverOpt)

// ServiceRegistrationConfig is the config passed to a service registration function
type ServiceRegistrationConfig struct {
	Config      *viper.Viper
	GrpcServer  *grpc.Server
	RestGateway *runtime.ServeMux
	Providers   providers.All
}

// Service is a an interface that registers a service with the server
type Service interface {
	// Register registers a service with the server
	Register(ctx context.Context, cfg ServiceRegistrationConfig) error
}

// ServiceRegistration is a function that registers a service with the server
type ServiceRegistration func(ctx context.Context, cfg ServiceRegistrationConfig) error

// Register implements the Service interface
func (s ServiceRegistration) Register(ctx context.Context, cfg ServiceRegistrationConfig) error {
	return s(ctx, cfg)
}

// WithUnaryInterceptors adds unary interceptors to the server
func WithUnaryInterceptors(interceptors ...grpc.UnaryServerInterceptor) ServerOption {
	return func(opt *serverOpt) {
		opt.UnaryInterceptors = append(opt.UnaryInterceptors, interceptors...)
	}
}

// WithStreamInterceptors adds interceptors to the grpc server
func WithStreamInterceptors(interceptors ...grpc.StreamServerInterceptor) ServerOption {
	return func(opt *serverOpt) {
		opt.StreamInterceptors = append(opt.StreamInterceptors, interceptors...)
	}
}

// WithGatewayOpts adds options to the grpc gateway
func WithGatewayOpts(opts ...runtime.ServeMuxOption) ServerOption {
	return func(opt *serverOpt) {
		opt.GatewayOpts = append(opt.GatewayOpts, opts...)
	}
}

// WithLogger adds a logging provider
func WithLogger(provider providers.LoggingProvider) ServerOption {
	return func(opt *serverOpt) {
		opt.Logger = provider
	}
}

// WithDatabase adds a database provider
func WithDatabase(provider providers.DatabaseProvider) ServerOption {
	return func(opt *serverOpt) {
		opt.Database = provider
	}
}

// WithCache adds a cache provider
func WithCache(provider providers.CacheProvider) ServerOption {
	return func(opt *serverOpt) {
		opt.Cache = provider
	}
}

// WithStream adds a stream provider
func WithStream(provider providers.StreamProvider) ServerOption {
	return func(opt *serverOpt) {
		opt.Stream = provider
	}
}

// WithContextTagger adds a context tagger to the server
func WithContextTagger(tagger providers.ContextTaggerProvider) ServerOption {
	return func(opt *serverOpt) {
		opt.Tagger = tagger
	}
}

// WithAuth adds an auth provider to the server (see github.com/autom8ter/protoc-gen-authenticate)
func WithAuth(auth grpc_auth.AuthFunc) ServerOption {
	return func(opt *serverOpt) {
		opt.Auth = auth
	}
}

// WithAuthz adds the authorizers to the server (see github.com/autom8ter/protoc-gen-authorize)
func WithAuthz(authorizers ...authorizer.Authorizer) ServerOption {
	return func(opt *serverOpt) {
		opt.Authz = authorizers
	}
}

// WithRateLimit adds a rate limiter to the server (see protoc-gen-ratelimit)
func WithRateLimit(rateLimit limiter.Limiter) ServerOption {
	return func(opt *serverOpt) {
		opt.RateLimit = rateLimit
	}
}

// CustomHTTPRoute is a custom route that can be added to the rest-gateway
type CustomHTTPRoute struct {
	// Method is the http method
	Method string
	// Path is the http path
	Path string
	// Handler is the http handler
	Handler runtime.HandlerFunc
}

// WithMetrics adds a metrics provider to the server
func WithMetrics(metrics providers.MetricsProvider) ServerOption {
	return func(opt *serverOpt) {
		opt.Metrics = metrics
		opt.Handlers = append(opt.Handlers, CustomHTTPRoute{
			Method: http.MethodGet,
			Path:   "/metrics",
			Handler: func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
				promhttp.Handler().ServeHTTP(w, r)
			},
		})
	}
}

// WithCustomHTTPRoute adds a custom http route to the rest-gateway
func WithCustomHTTPRoute(method, path string, handler runtime.HandlerFunc) ServerOption {
	return func(opt *serverOpt) {
		opt.Handlers = append(opt.Handlers, CustomHTTPRoute{
			Method:  method,
			Path:    path,
			Handler: handler,
		})
	}
}

// WithGrpcHealthCheck adds a grpc health check to the server
func WithGrpcHealthCheck(srv grpc_health_v1.HealthServer) ServerOption {
	return func(opt *serverOpt) {
		opt.GrpcHealthCheck = srv
	}
}

// Server is a highly configurable grpc server with a built-in rest-gateway(grpc-gateway)
// The server supports the following features via ServerOptions:
// - Logging Interface (slog)
// - Metrics Interface (prometheus)
// - Database Interface (sqlite/mysql/postgres)
// - Cache Interface (redis)
// - Stream Interface (nats/redis)
// - Context Tags
// - Authentication (see github.com/autom8ter/protoc-gen-authenticate)
// - Authorization (see github.com/autom8ter/protoc-gen-authorize)
// - Rate Limiting (see github.com/autom8ter/protoc-gen-ratelimit)
type Server struct {
	cfg         *viper.Viper
	health      grpc_health_v1.HealthServer
	providers   providers.All
	grpcOpts    []grpc.ServerOption
	gatewayOpts []runtime.ServeMuxOption
	httpRoutes  []CustomHTTPRoute
}

// NewServer creates a new server with the given config and options
func NewServer(ctx context.Context, cfg *viper.Viper, opts ...ServerOption) (*Server, error) {
	s := &Server{
		cfg: cfg,
	}
	var sopts = &serverOpt{}
	for _, opt := range opts {
		opt(sopts)
	}
	if sopts.Logger == nil {
		sopts.Logger = slog2.Provider
	}
	if sopts.Tagger == nil {
		sopts.Tagger = maptags.Provider
	}
	log, err := sopts.Logger(ctx, cfg)
	if err != nil {
		return nil, utils.WrapError(err, "failed to create logger")
	}
	prviders := providers.All{
		Logger: log,
	}

	if sopts.Database != nil {
		db, err := sopts.Database(ctx, cfg)
		if err != nil {
			return nil, utils.WrapError(err, "failed to create database")
		}
		// create schema
		if cfg.GetBool("database.migrate") {
			if err := db.Migrate(ctx); err != nil {
				return nil, utils.WrapError(err, "failed to migrate database")
			}
		}
		prviders.Database = db
	}
	if sopts.Cache != nil {
		cahe, err := sopts.Cache(ctx, cfg)
		if err != nil {
			return nil, utils.WrapError(err, "failed to create cache")
		}
		prviders.Cache = cahe
	}
	if sopts.Stream != nil {
		que, err := sopts.Stream(ctx, cfg)
		if err != nil {
			return nil, utils.WrapError(err, "failed to create stream")
		}
		prviders.Stream = que
	}

	{
		tags, err := sopts.Tagger(ctx, cfg)
		if err != nil {
			return nil, utils.WrapError(err, "failed to create context tagger")
		}
		// context_tagger interceptor must be first
		sopts.UnaryInterceptors = append([]grpc.UnaryServerInterceptor{providers.UnaryContextTaggerInterceptor(tags)}, sopts.UnaryInterceptors...)
		sopts.StreamInterceptors = append([]grpc.StreamServerInterceptor{providers.StreamContextTaggerInterceptor(tags)}, sopts.StreamInterceptors...)

		if sopts.Metrics != nil {
			metrics, err := sopts.Metrics(ctx, cfg)
			if err != nil {
				return nil, utils.WrapError(err, "failed to create metrics")
			}
			prviders.Metrics = metrics
			sopts.UnaryInterceptors = append(sopts.UnaryInterceptors, providers.UnaryMetricsInterceptor(metrics))
			sopts.StreamInterceptors = append(sopts.StreamInterceptors, providers.StreamMetricsInterceptor(metrics))
		}

		sopts.UnaryInterceptors = append(sopts.UnaryInterceptors, providers.UnaryLoggingInterceptor(cfg.GetBool("logging.request_body"), log))
		sopts.StreamInterceptors = append(sopts.StreamInterceptors, providers.StreamLoggingInterceptor(cfg.GetBool("logging.request_body"), log))
		if sopts.RateLimit != nil {
			sopts.UnaryInterceptors = append(sopts.UnaryInterceptors, limiter.UnaryServerInterceptor(sopts.RateLimit))
			sopts.StreamInterceptors = append(sopts.StreamInterceptors, limiter.StreamServerInterceptor(sopts.RateLimit))
		}
		if sopts.Auth != nil {
			sopts.UnaryInterceptors = append(sopts.UnaryInterceptors, grpc_auth.UnaryServerInterceptor(sopts.Auth))
			sopts.StreamInterceptors = append(sopts.StreamInterceptors, grpc_auth.StreamServerInterceptor(sopts.Auth))
		}
		if len(sopts.Authz) > 0 {
			sopts.UnaryInterceptors = append(sopts.UnaryInterceptors, authorizer.UnaryServerInterceptor(sopts.Authz))
			sopts.StreamInterceptors = append(sopts.StreamInterceptors, authorizer.StreamServerInterceptor(sopts.Authz))
		}

		sopts.UnaryInterceptors = append(sopts.UnaryInterceptors, grpc_validator.UnaryServerInterceptor())
		sopts.StreamInterceptors = append(sopts.StreamInterceptors, grpc_validator.StreamServerInterceptor())

		sopts.StreamInterceptors = append(sopts.StreamInterceptors, grpc_recovery.StreamServerInterceptor())
		sopts.UnaryInterceptors = append(sopts.UnaryInterceptors, grpc_recovery.UnaryServerInterceptor())
	}

	var grpcOpts []grpc.ServerOption

	grpcOpts = append(grpcOpts,
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(sopts.UnaryInterceptors...)),
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(sopts.StreamInterceptors...)),
	)
	s.grpcOpts = grpcOpts
	s.gatewayOpts = sopts.GatewayOpts
	s.providers = prviders
	s.httpRoutes = sopts.Handlers
	s.health = sopts.GrpcHealthCheck
	return s, nil
}

// Providers returns the server providers
func (s *Server) Providers() providers.All {
	return s.providers
}

// Config returns the server config
func (s *Server) Config() *viper.Viper {
	return s.cfg
}

// Serve registers the given services and starts the server. This function blocks until the server is shutdown.
// The server will shutdown when the context is canceled or an interrupt signal is received.
// The server will start grpc/rest-gateway servers on the port specified by the config key "api.port"
// The server will register a health check at /health and a readiness check at /ready
// The server will register a metrics endpoint at /metrics if the config key "metrics.prometheus" is true
func (s *Server) Serve(ctx context.Context, services ...Service) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	srv := grpc.NewServer(
		s.grpcOpts...,
	)
	gwMux := runtime.NewServeMux(s.gatewayOpts...)
	for _, handler := range s.httpRoutes {
		if err := gwMux.HandlePath(handler.Method, handler.Path, handler.Handler); err != nil {
			return utils.WrapError(err, "failed to register custom http route")
		}
	}
	serviceConfig := ServiceRegistrationConfig{
		Config:      s.cfg,
		GrpcServer:  srv,
		RestGateway: gwMux,
		Providers:   s.providers,
	}
	for _, service := range services {
		if err := service.Register(ctx, serviceConfig); err != nil {
			return utils.WrapError(err, "failed to register service")
		}
	}
	if s.health != nil {
		grpc_health_v1.RegisterHealthServer(srv, s.health)
	}
	lis, err := net.Listen("tcp", fmt.Sprintf(":%v", s.cfg.GetInt("api.port")))
	if err != nil {
		return utils.WrapError(err, "failed to listen on port %v", s.cfg.GetInt("api.port"))
	}
	defer lis.Close()

	m := cmux.New(lis)
	defer m.Close()
	grpcMatcher := m.MatchWithWriters(cmux.HTTP2MatchHeaderFieldSendSettings("content-type", "application/grpc"))
	restMatcher := m.Match(cmux.HTTP1Fast())
	var mux http.Handler
	if s.cfg.GetBool("api.cors.enabled") {
		mux = cors.New(cors.Options{
			AllowedOrigins:   s.cfg.GetStringSlice("api.cors.allowed_origins"),
			AllowedMethods:   s.cfg.GetStringSlice("api.cors.allowed_methods"),
			AllowedHeaders:   s.cfg.GetStringSlice("api.cors.allowed_headers"),
			ExposedHeaders:   s.cfg.GetStringSlice("api.cors.exposed_headers"),
			AllowCredentials: s.cfg.GetBool("api.cors.allow_credentials"),
		}).Handler(gwMux)
	} else {
		mux = gwMux
	}
	server := &http.Server{Handler: mux}
	port := s.cfg.GetInt("api.port")
	egp, ctx := errgroup.WithContext(ctx)
	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
		<-ch
		s.providers.Logger.Info(ctx, "interrupt signal received: shutting down servers")
		cancel()
	}()
	go func() {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		<-ctx.Done()
		s.providers.Logger.Debug(ctx, "context canceled: shutting down servers")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
		srv.GracefulStop()
		m.Close()
	}()
	egp.Go(func() error {
		s.providers.Logger.Info(ctx, "starting grpc server", map[string]any{
			"port": port,
		})
		return utils.WrapError(srv.Serve(grpcMatcher), "")
	})
	egp.Go(func() error {
		s.providers.Logger.Info(ctx, "starting rest server", map[string]any{
			"port": port,
		})
		return utils.WrapError(server.Serve(restMatcher), "")
	})
	egp.Go(func() error {
		m.Serve()
		return nil
	})
	if err := egp.Wait(); err != nil && isServerFailure(err) {
		return utils.WrapError(err, "failed to serve")
	}
	return nil
}

func isServerFailure(err error) bool {
	if err == nil {
		return false
	}
	if stacktrace.RootCause(err) == http.ErrServerClosed {
		return false
	}
	if stacktrace.RootCause(err) == grpc.ErrServerStopped {
		return false
	}
	if stacktrace.RootCause(err) == context.Canceled {
		return false
	}
	if strings.Contains(err.Error(), "mux: server closed") {
		return false
	}
	return true
}
