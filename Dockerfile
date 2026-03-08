FROM dhi.io/golang:1.26.1-alpine3.23-dev@sha256:55d6cb7d99b2e6e6ed9a766a0b9e39f6cf29fa0471c2f274d90102d758c8e0b3 AS builder

ARG VERSION=0.0.0

# Set the current working directory inside the container.
WORKDIR /go/src/github.com/score-spec/score-compose

# Copy just the module bits
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire project and build it.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w -X github.com/score-spec/score-compose/internal/version.Version=${VERSION}" -o /usr/local/bin/score-compose ./cmd/score-compose

# We can use static since we don't rely on any linux libs or state, but we need ca-certificates to connect to https/oci with the init command.
FROM dhi.io/static:20251003-alpine3.23@sha256:a08d9a53a4758b4006d56341aa88b1edf583ddebd93e620a32acd5135535573c

# Set the current working directory inside the container.
WORKDIR /score-compose

# Copy the binary from the builder image.
COPY --from=builder /usr/local/bin/score-compose /usr/local/bin/score-compose

# Run the binary.
ENTRYPOINT ["/usr/local/bin/score-compose"]
