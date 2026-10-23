FROM dhi.io/golang:1.27.3-alpine3.24-dev@sha256:2ceb813d69b9c82501323484e0c0ae612efde281f5dffce2a3d83a79a9c87304 AS builder

ARG VERSION=0.0.0
ARG GIT_COMMIT=unknown
ARG BUILD_DATE=unknown

# Set the current working directory inside the container.
WORKDIR /go/src/github.com/score-spec/score-compose

# Copy just the module bits
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire project and build it.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux \
    go build -ldflags="-s -w \
        -X github.com/score-spec/score-compose/internal/version.Version=${VERSION} \
        -X github.com/score-spec/score-compose/internal/version.GitCommit=${GIT_COMMIT} \
        -X github.com/score-spec/score-compose/internal/version.BuildDate=${BUILD_DATE}" \
    -o /usr/local/bin/score-compose ./cmd/score-compose

# We can use static since we don't rely on any linux libs or state, but we need ca-certificates to connect to https/oci with the init command.
FROM dhi.io/static:20261020-alpine3.24@sha256:c78b2c2034b3fb2f83a337b6353c3ef27d910a828d9127ff342014e7b225c04e

# Set the current working directory inside the container.
WORKDIR /score-compose

# Copy the binary from the builder image.
COPY --from=builder /usr/local/bin/score-compose /usr/local/bin/score-compose

# Run the binary.
ENTRYPOINT ["/usr/local/bin/score-compose"]
