# syntax=docker/dockerfile:1
# The Go toolchain from the Docker Hardened Image for Go, the same image
# family the score-compose Dockerfile builds with, copied onto scratch so
# the overlay lands on any base. dhi.io/golang keeps Go in /usr/lib/go; the
# shell workload already has /usr/local/go/bin on its PATH.
ARG GO_VERSION
FROM dhi.io/golang:${GO_VERSION} AS go

FROM scratch
COPY --from=go /usr/lib/go /usr/local/go
ENTRYPOINT ["/usr/local/go/bin/go"]
CMD ["version"]
