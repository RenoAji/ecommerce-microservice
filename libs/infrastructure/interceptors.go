package infrastructure

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func newClientInterceptors(secret string) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if corID, ok := ctx.Value("correlation_id").(string); ok && corID != "" {
			ctx = metadata.AppendToOutgoingContext(ctx, "x-correlation-id", corID)
		}

		ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+secret)
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
