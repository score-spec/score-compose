FROM dhi.io/golang:1.26.4-alpine3.24-dev@sha256:8c0a72d824e41949727f0907f851fff7bbd26b788d8e116facdddf84cb9f2905 AS builder

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
FROM dhi.io/static:20260611-alpine3.24@sha256:390fea8b496568bd8e8f085ab8a1c92403d9baa047e1f82436c7874694de2c2d

# Set the current working directory inside the container.
WORKDIR /score-compose

# Copy the binary from the builder image.
COPY --from=builder /usr/local/bin/score-compose /usr/local/bin/score-compose

# Run the binary.
ENTRYPOINT ["/usr/local/bin/score-compose"]
