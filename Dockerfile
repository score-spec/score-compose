FROM dhi.io/golang:1.26.5-alpine3.24-dev@sha256:29fe0a7d2a5ab0c236fbde3a7f63801755585ed260b6f2f564e831c92bfa9f34 AS builder

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
FROM dhi.io/static:20260611-alpine3.24@sha256:93568eb7c673afb3ad79b15cca341469d3e02cf859caae1049aa22fe7fbce90a

# Set the current working directory inside the container.
WORKDIR /score-compose

# Copy the binary from the builder image.
COPY --from=builder /usr/local/bin/score-compose /usr/local/bin/score-compose

# Run the binary.
ENTRYPOINT ["/usr/local/bin/score-compose"]
