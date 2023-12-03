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

	"github.com/autom8ter/protoc-gen-authenticate/authenticator"
	"github.com/autom8ter/protoc-gen-authorize/authorizer"
	"github.com/autom8ter/protoc-gen-ratelimit/limiter"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/selector"
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

type authWithSelectors struct {
	authenticator.Authenticator
	selectors []selector.Matcher
}

type rateLimitWithSelectors struct {
	limiter.Limiter
	selectors []selector.Matcher
}

type authzWithOptions struct {
	authorizer.Authorizer
	options []authorizer.Opt
}

type serverOpt struct {
	GatewayOpts        []runtime.ServeMuxOption
	UnaryInterceptors  []grpc.UnaryServerInterceptor
	StreamInterceptors []grpc.StreamServerInterceptor
	Logger             providers.Logger
	Database           providers.Database
	Cache              providers.Cache
	Stream             providers.Stream
	Email              providers.Emailer
	Auth               []authWithSelectors
	Authz              []authzWithOptions
	Tagger             providers.ContextTagger
	RateLimit          []rateLimitWithSelectors
	Metrics            providers.Metrics
	Handlers           []CustomHTTPRoute
	GrpcHealthCheck    grpc_health_v1.HealthServer
	Validation         bool
	PaymentProcessor   providers.PaymentProcessor
	Storage            providers.Storage
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

// WithValidation enables grpc validation interceptors for validating requests
func WithValidation() ServerOption {
	return func(opt *serverOpt) {
		opt.Validation = true
	}
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
func WithLogger(provider providers.Logger) ServerOption {
	return func(opt *serverOpt) {
		opt.Logger = provider
	}
}

// WithEmail adds an email provider
func WithEmail(provider providers.Emailer) ServerOption {
	return func(opt *serverOpt) {
		opt.Email = provider
	}
}

// WithDatabase adds a database provider
func WithDatabase(provider providers.Database) ServerOption {
	return func(opt *serverOpt) {
		opt.Database = provider
	}
}

// WithStorage adds a storage provider
func WithStorage(provider providers.Storage) ServerOption {
	return func(opt *serverOpt) {
		opt.Storage = provider
	}
}

// WithCache adds a cache provider
func WithCache(provider providers.Cache) ServerOption {
	return func(opt *serverOpt) {
		opt.Cache = provider
	}
}

// WithStream adds a stream provider
func WithStream(provider providers.Stream) ServerOption {
	return func(opt *serverOpt) {
		opt.Stream = provider
	}
}

// WithContextTagger adds a context tagger to the server
func WithContextTagger(tagger providers.ContextTagger) ServerOption {
	return func(opt *serverOpt) {
		opt.Tagger = tagger
	}
}

// WithAuth adds an auth provider to the server (see github.com/autom8ter/protoc-gen-authenticate)
// If no selectors are provided, the auth provider will be used for all requests
// This method can be called multiple times to add multiple auth providers
func WithAuth(auth authenticator.AuthFunc, selectors ...selector.Matcher) ServerOption {
	return func(opt *serverOpt) {
		opt.Auth = append(opt.Auth, authWithSelectors{
			Authenticator: auth,
			selectors:     selectors,
		})
	}
}

// WithAuthz adds an authorizer to the server (see github.com/autom8ter/protoc-gen-authorize)
// This method can be called multiple times to add multiple authorizers
// Options can be added to add a userExtractor and selectors
func WithAuthz(authorizer authorizer.Authorizer, opts ...authorizer.Opt) ServerOption {
	return func(opt *serverOpt) {
		opt.Authz = append(opt.Authz, authzWithOptions{
			Authorizer: authorizer,
			options:    opts,
		})
	}
}

// WithRateLimit adds a rate limiter to the server (see protoc-gen-ratelimit)
// If no selectors are provided, the rate limiter will be used for all requests
// This method can be called multiple times to add multiple rate limiters
func WithRateLimit(rateLimit limiter.Limiter, selectors ...selector.Matcher) ServerOption {
	return func(opt *serverOpt) {
		opt.RateLimit = append(opt.RateLimit, rateLimitWithSelectors{
			Limiter:   rateLimit,
			selectors: selectors,
		})
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
func WithMetrics(metrics providers.Metrics) ServerOption {
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

// WithPaymentProcessor adds a payment processor to the server
func WithPaymentProcessor(processor providers.PaymentProcessor) ServerOption {
	return func(opt *serverOpt) {
		opt.PaymentProcessor = processor
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
		lgger, err := slog2.Provider(ctx, cfg)
		if err != nil {
			return nil, utils.WrapError(err, "failed to create logger")
		}
		sopts.Logger = lgger
		sopts.Logger.Debug(ctx, "registered default logger")
	}
	if sopts.Tagger == nil {
		tgger, err := maptags.Provider(ctx, cfg)
		if err != nil {
			return nil, utils.WrapError(err, "failed to create tagger")
		}
		sopts.Tagger = tgger
		sopts.Logger.Debug(ctx, "registered default context tagger")
	}

	prviders := providers.All{
		Logger:           sopts.Logger,
		Email:            sopts.Email,
		Database:         sopts.Database,
		Cache:            sopts.Cache,
		Stream:           sopts.Stream,
		PaymentProcessor: sopts.PaymentProcessor,
		Storage:          sopts.Storage,
	}

	{
		// context_tagger interceptor must be first
		sopts.UnaryInterceptors = append([]grpc.UnaryServerInterceptor{providers.UnaryContextTaggerInterceptor(sopts.Tagger)}, sopts.UnaryInterceptors...)
		sopts.StreamInterceptors = append([]grpc.StreamServerInterceptor{providers.StreamContextTaggerInterceptor(sopts.Tagger)}, sopts.StreamInterceptors...)

		if sopts.Metrics != nil {
			sopts.UnaryInterceptors = append(sopts.UnaryInterceptors, providers.UnaryMetricsInterceptor(sopts.Metrics))
			sopts.StreamInterceptors = append(sopts.StreamInterceptors, providers.StreamMetricsInterceptor(sopts.Metrics))
			sopts.Logger.Debug(ctx, "registered metrics provider")
		}

		sopts.UnaryInterceptors = append(sopts.UnaryInterceptors, providers.UnaryLoggingInterceptor(cfg.GetBool("logging.request_body"), sopts.Logger))
		sopts.StreamInterceptors = append(sopts.StreamInterceptors, providers.StreamLoggingInterceptor(cfg.GetBool("logging.request_body"), sopts.Logger))
		if len(sopts.RateLimit) > 0 {
			for _, rl := range sopts.RateLimit {
				sopts.UnaryInterceptors = append(sopts.UnaryInterceptors, limiter.UnaryServerInterceptor(rl.Limiter, rl.selectors...))
				sopts.StreamInterceptors = append(sopts.StreamInterceptors, limiter.StreamServerInterceptor(rl.Limiter, rl.selectors...))
			}
			sopts.Logger.Debug(ctx, "registered rate limiter(s)")
		}
		if len(sopts.Auth) > 0 {
			for _, ath := range sopts.Auth {
				sopts.UnaryInterceptors = append(sopts.UnaryInterceptors, authenticator.UnaryServerInterceptor(ath, ath.selectors...))
				sopts.StreamInterceptors = append(sopts.StreamInterceptors, authenticator.StreamServerInterceptor(ath, ath.selectors...))
			}
			sopts.Logger.Debug(ctx, "registered authenticator(s)")
		}
		if len(sopts.Authz) > 0 {
			for _, athz := range sopts.Authz {
				sopts.UnaryInterceptors = append(sopts.UnaryInterceptors, authorizer.UnaryServerInterceptor(athz, athz.options...))
				sopts.StreamInterceptors = append(sopts.StreamInterceptors, authorizer.StreamServerInterceptor(athz, athz.options...))
			}
			sopts.Logger.Debug(ctx, "registered authorizer(s)")
		}
		if sopts.Validation {
			sopts.UnaryInterceptors = append(sopts.UnaryInterceptors, grpc_validator.UnaryServerInterceptor())
			sopts.StreamInterceptors = append(sopts.StreamInterceptors, grpc_validator.StreamServerInterceptor())
			sopts.Logger.Debug(ctx, "registered validation interceptor")
		}

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
		s.providers.Logger.Debug(ctx, "registered custom http route", map[string]any{
			"method": handler.Method,
			"path":   handler.Path,
		})
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
		s.providers.Logger.Debug(ctx, "registered grpc health check")
	}
	if s.cfg.GetBool("database.migrate") {
		s.providers.Logger.Debug(ctx, "performing database migration...")
		if err := s.providers.Database.Migrate(ctx); err != nil {
			return utils.WrapError(err, "failed to migrate database")
		}
		s.providers.Logger.Debug(ctx, "performed database migration")
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
		s.providers.Logger.Debug(ctx, "registered cors middleware")
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
