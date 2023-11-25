package grpcx_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/autom8ter/grpcx"
	echov1 "github.com/autom8ter/grpcx/gen/echo"
	"github.com/autom8ter/grpcx/grpcxtest"
	"github.com/autom8ter/grpcx/providers/basicauth"
	"github.com/autom8ter/grpcx/providers/maptags"
	redis2 "github.com/autom8ter/grpcx/providers/redis"
	slog2 "github.com/autom8ter/grpcx/providers/slog"
	"github.com/autom8ter/grpcx/providers/sqlite"
)

var fixtures = []*grpcxtest.Fixture{
	{
		Config:     viper.New(),
		Timeout:    30 * time.Second,
		ServerOpts: nil,
		Services: []grpcx.Service{
			grpcx.EchoService(),
		},
		Name: "testing",
		Test: func(t *testing.T, ctx context.Context, client grpc.ClientConnInterface) {
			t.Log("testing")
		},
		ClientMeta: map[string]string{},
	},
	{
		Config:     viper.New(),
		Timeout:    30 * time.Second,
		ServerOpts: nil,
		Services:   []grpcx.Service{grpcx.EchoService()},
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

func Test(t *testing.T) {
	for _, fixture := range fixtures {
		fixture.RunTest(t)
	}
}

func ExampleNewServer() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cfg, err := grpcx.NewConfig()
	if err != nil {
		panic(err)
	}
	cfg.Set("auth.username", "test")
	cfg.Set("auth.password", "test")
	srv, err := grpcx.NewServer(
		ctx,
		cfg,
		// Register Auth provider
		grpcx.WithAuth(basicauth.New()),
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
	)
	if err != nil {
		panic(err)
	}
	go func() {
		if err := srv.Serve(ctx, grpcx.EchoService()); err != nil {
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
