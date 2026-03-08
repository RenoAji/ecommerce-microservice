# Stage 1: Build the binary
FROM golang:1.25-alpine AS builder

# Install git (sometimes needed for fetching dependencies)
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy dependency files first (for better caching)
COPY go.mod go.sum ./
COPY --from=libs ./consulclient/go.mod ./consulclient/go.sum ../libs/consulclient/
COPY --from=libs ./infrastructure/go.mod ./infrastructure/go.sum ../libs/infrastructure/
COPY --from=libs ./pb/go.mod ./pb/go.sum ../libs/pb/
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
 go mod download
COPY --from=libs . ../libs

# Copy the rest of the source code
COPY --from=libs . ../libs
COPY . .

# Build the app (pointing to your main.go location)
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/main ./cmd/api/main.go

# Stage 2: Final lightweight image
FROM alpine:latest

WORKDIR /root/

# Copy only the compiled binary from the builder stage
COPY --from=builder /app/main .
# Copy .env if you use one (optional)
# COPY .env .

# Expose the port your app runs on
EXPOSE 8081

# Command to run
CMD ["./main"]