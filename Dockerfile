FROM dhi.io/golang:1.25.6-alpine3.23-dev@sha256:31ff552fcbb92889bcf952a4b71c1ae71c84e5b067bfe78038217427e905330e AS builder

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
FROM dhi.io/static:20251003-alpine3.23@sha256:f5c9799d5834f66db6912303ba88eaf53a3c7cc3ddc97a282a6bbd8672e15d24

# Set the current working directory inside the container.
WORKDIR /score-compose

# Copy the binary from the builder image.
COPY --from=builder /usr/local/bin/score-compose /usr/local/bin/score-compose

# Run the binary.
ENTRYPOINT ["/usr/local/bin/score-compose"]
