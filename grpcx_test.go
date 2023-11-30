package grpcx_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/autom8ter/grpcx"
	echov1 "github.com/autom8ter/grpcx/gen/echo"
	"github.com/autom8ter/grpcx/grpcxtest"
	"github.com/autom8ter/grpcx/providers/maptags"
	"github.com/autom8ter/grpcx/providers/prometheus"
	redis2 "github.com/autom8ter/grpcx/providers/redis"
	slog2 "github.com/autom8ter/grpcx/providers/slog"
	"github.com/autom8ter/grpcx/providers/sqlite"
)

func fixtures() []*grpcxtest.Fixture {
	cfg, err := grpcx.LoadConfig("test-api", "", "TEST_API")
	if err != nil {
		panic(err)
	}
	cfg.Set("database.connection_string", "file::memory:?cache=shared")
	return []*grpcxtest.Fixture{
		{
			Config:  cfg,
			Timeout: 30 * time.Second,
			ServerOpts: []grpcx.ServerOption{
				grpcx.WithLogger(slog2.Provider),
				grpcx.WithContextTagger(maptags.Provider),
				grpcx.WithCache(redis2.InMemProvider),
				grpcx.WithDatabase(sqlite.Provider),
				grpcx.WithStream(redis2.InMemStreamProvider),
				grpcx.WithMetrics(prometheus.Provider),
			},
			Services: []grpcx.Service{
				EchoService(),
			},
			Name: "testing",
			Test: func(t *testing.T, ctx context.Context, client grpc.ClientConnInterface) {
				t.Log("testing")
			},
			ClientMeta: map[string]string{},
		},
		{
			Config:     cfg,
			Timeout:    30 * time.Second,
			ServerOpts: nil,
			Services:   []grpcx.Service{EchoService()},
			Name:       "echo",
			Test: func(t *testing.T, ctx context.Context, client grpc.ClientConnInterface) {
				echoClient := echov1.NewEchoServiceClient(client)
				resp, err := echoClient.Echo(ctx, &echov1.EchoRequest{
					Message: "hello",
				})
				require.NoError(t, err)
				require.Equal(t, "hello", resp.Message)
				require.Equal(t, "test", resp.ClientMetadata["x-test"], "%v", resp.ClientMetadata)
			},
			ClientMeta: map[string]string{
				"X-Test": "test",
			},
		},
	}
}

func Test(t *testing.T) {
	for _, fixture := range fixtures() {
		fixture.RunTest(t)
	}
}

func ExampleNewServer() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cfg, err := grpcx.LoadConfig("test-api", "", "TEST_API")
	if err != nil {
		panic(err)
	}
	cfg.Set("auth.username", "test")
	cfg.Set("auth.password", "test")
	srv, err := grpcx.NewServer(
		ctx,
		cfg,
		// Register Cache Provider
		grpcx.WithCache(redis2.InMemProvider),
		// Register Stream Provider
		grpcx.WithStream(redis2.InMemStreamProvider),
		// Register Context Tagger
		grpcx.WithContextTagger(maptags.Provider),
		// Register Logger
		grpcx.WithLogger(slog2.Provider),
		// Register Database
		grpcx.WithDatabase(sqlite.Provider),
		// Register Metrics
		grpcx.WithMetrics(prometheus.Provider),
	)
	if err != nil {
		panic(err)
	}
	go func() {
		if err := srv.Serve(ctx, EchoService()); err != nil {
			panic(err)
		}
	}()

	conn, err := grpc.DialContext(ctx, fmt.Sprintf("localhost:%v", cfg.GetInt("api.port")), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	echoClient := echov1.NewEchoServiceClient(conn)
	ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "test:test")
	resp, err := echoClient.Echo(ctx, &echov1.EchoRequest{
		Message: "hello",
	})
	if err != nil {
		panic(err)
	}
	println(resp.Message)
}

type echoServer struct {
	echov1.UnimplementedEchoServiceServer
}

func (e *echoServer) Echo(ctx context.Context, req *echov1.EchoRequest) (*echov1.EchoResponse, error) {
	var meta = map[string]string{}
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "no metadata found in context")
	}
	for k, v := range md {
		meta[k] = v[0]
	}
	return &echov1.EchoResponse{
		Message:        req.Message,
		ClientMetadata: meta,
	}, nil
}

// EchoService returns a ServiceRegistration that registers an echo service
func EchoService() grpcx.ServiceRegistration {
	return grpcx.ServiceRegistration(func(ctx context.Context, cfg grpcx.ServiceRegistrationConfig) error {
		echov1.RegisterEchoServiceServer(cfg.GrpcServer, &echoServer{})
		return nil
	})
}
