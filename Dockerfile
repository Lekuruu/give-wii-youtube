# syntax=docker/dockerfile:1.7

FROM golang:1.25-alpine AS build

WORKDIR /app

# Copy module files
COPY ./go.mod .
COPY ./go.sum .

# Download dependencies (cached between builds)
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy the source code
COPY . .

# Build the server with cached build artifacts
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 go build -o give-wii-youtube ./cmd/api/

FROM alpine AS app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates ffmpeg

WORKDIR /app
COPY --from=build /app/give-wii-youtube /app/give-wii-youtube

# Create static directory volume
VOLUME ["/app/static"]

# Create data volume
VOLUME ["/app/data"]

# Run the compiled binary
ENTRYPOINT ["/app/give-wii-youtube"]