//go:generate mockgen -destination=./mocks/context_tagger.go -package=mocks . ContextTagger

package providers

import (
	"context"

	"github.com/google/uuid"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type ctxKey string

const (
	tagsKey ctxKey = "tags"
)

// Tags is an interface for tagging contexts
type Tags interface {
	// LogTags returns the tags for logging
	LogTags() map[string]any
	// WithMetadata adds metadata to the tags
	WithMetadata(md metadata.MD) Tags
	// WithMethod adds the grpc method to the tags
	WithMethod(method string) Tags
	// WithContextID adds the contextID to the tags
	WithContextID(contextID string) Tags
	// WithError adds the error to the tags
	WithError(err error) Tags
	// GetMetadata returns the metadata from the tags
	GetMetadata() (metadata.MD, bool)
	// GetMethod returns the grpc method from the tags
	GetMethod() (string, bool)
	// GetContextID returns the contextID from the tags
	GetContextID() (string, bool)
	// GetError returns the error from the tags
	GetError() (error, bool)
}

// GetTags returns the tags from the context
func GetTags(ctx context.Context) (Tags, bool) {
	tags, ok := ctx.Value(tagsKey).(Tags)
	if !ok {
		return nil, false
	}
	return tags, true
}

// WithTags adds tags to the context
func WithTags(ctx context.Context, tags Tags) context.Context {
	return context.WithValue(ctx, tagsKey, tags)
}

// ContextTagger is an interface for tagging contexts
type ContextTagger interface {
	// TagContext tags the context with the given contextID and grpcMethod
	TagContext(ctx context.Context) Tags
}

// ContextTaggerFunc is a function that implements the ContextTagger interface
type ContextTaggerFunc func(ctx context.Context) Tags

// TagContext tags the context with the given contextID and grpcMethod
func (c ContextTaggerFunc) TagContext(ctx context.Context) Tags {
	return c(ctx)
}

// ContextTaggerProvider is a function that returns a ContextTagger
type ContextTaggerProvider func(ctx context.Context, cfg *viper.Viper) (ContextTagger, error)

// UnaryContextTaggerInterceptor is a grpc unary interceptor that tags the inbound context
func UnaryContextTaggerInterceptor(tagger ContextTagger) func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		tags := tagger.TagContext(ctx)
		var (
			method    = info.FullMethod
			contextID string
		)
		grpcMetadata, ok := metadata.FromIncomingContext(ctx)
		if ok {
			id := grpcMetadata.Get("x-context-id")
			if len(id) > 0 {
				contextID = id[0]
			} else {
				contextID = uuid.NewString()
			}
		}
		tags = tags.
			WithContextID(contextID).
			WithMethod(method).
			WithMetadata(grpcMetadata)
		return handler(WithTags(ctx, tags), req)
	}
}

// StreamContextTaggerInterceptor is a grpc stream interceptor that tags the inbound context
func StreamContextTaggerInterceptor(tagger ContextTagger) func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		tags := tagger.TagContext(ss.Context())
		var (
			method    = info.FullMethod
			contextID string
		)
		grpcMetadata, ok := metadata.FromIncomingContext(ss.Context())
		if ok {
			id := grpcMetadata.Get("X-Context-Id")
			if len(id) > 0 {
				contextID = id[0]
			} else {
				contextID = uuid.NewString()
			}
		}
		tags = tags.
			WithContextID(contextID).
			WithMethod(method).
			WithMetadata(grpcMetadata)
		return handler(srv, &serverStreamWrapper{
			ServerStream: ss,
			ctx:          WithTags(ss.Context(), tags),
		})
	}
}

type serverStreamWrapper struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *serverStreamWrapper) Context() context.Context {
	return s.ctx
}
