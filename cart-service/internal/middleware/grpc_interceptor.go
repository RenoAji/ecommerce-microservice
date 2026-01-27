package middleware

import (
	"context"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func InternalAuthInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
    md, _ := metadata.FromIncomingContext(ctx)
    
    // Get the secret from the service's own environment
    systemSecret := os.Getenv("INTERNAL_SECRET")
    
    token := md["authorization"]
    if len(token) == 0 || token[0] != "Bearer "+systemSecret {
        return nil, status.Error(codes.Unauthenticated, "invalid system token")
    }

    return handler(ctx, req)
}