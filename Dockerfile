FROM dhi.io/golang:1.25.5-alpine3.22@sha256:0ce472de71c3d85a8762a1662fa8030a504341fc61a97d51716c0afeb8481b9d AS builder

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
FROM dhi.io/static:20250911-alpine3.22@sha256:d36af7c8b762bca5ac335571b74a84887b11334e227e70cee0c17521edef5c13

# Set the current working directory inside the container.
WORKDIR /score-compose

# Copy the binary from the builder image.
COPY --from=builder /usr/local/bin/score-compose /usr/local/bin/score-compose

# Run the binary.
ENTRYPOINT ["/usr/local/bin/score-compose"]
