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
	"github.com/autom8ter/grpcx/providers/prometheus"
	redis2 "github.com/autom8ter/grpcx/providers/redis"
	"github.com/autom8ter/grpcx/providers/sqlite"
)

func fixtures() []*grpcxtest.Fixture {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cache, err := redis2.NewInMem()
	if err != nil {
		panic(err)
	}
	metrics := prometheus.New()
	database, err := sqlite.New(ctx, "file::memory:?cache=shared", "")
	if err != nil {
		panic(err)
	}
	return []*grpcxtest.Fixture{
		{
			Timeout: 30 * time.Second,
			ServerOpts: []grpcx.ServerOption{
				grpcx.WithCache(cache),
				grpcx.WithDatabase(database),
				grpcx.WithStream(cache),
				grpcx.WithMetrics(metrics),
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
	cache, err := redis2.NewInMem()
	if err != nil {
		panic(err)
	}
	metrics := prometheus.New()
	database, err := sqlite.New(ctx, "file::memory:?cache=shared", "")
	if err != nil {
		panic(err)
	}
	port := 8080
	if err != nil {
		panic(err)
	}
	srv, err := grpcx.NewServer(
		ctx,
		// Register Cache Provider
		grpcx.WithCache(cache),
		// Register Stream Provider
		grpcx.WithStream(cache),
		// Register Database
		grpcx.WithDatabase(database),
		// Register Metrics
		grpcx.WithMetrics(metrics),
	)
	if err != nil {
		panic(err)
	}
	go func() {
		if err := srv.Serve(ctx, port, EchoService()); err != nil {
			panic(err)
		}
	}()

	conn, err := grpc.DialContext(ctx, fmt.Sprintf("localhost:%v", port), grpc.WithInsecure())
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
