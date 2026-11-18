# syntax=docker/dockerfile:1
# The Go release tarball, checksum-verified, staged under /out and copied
# onto scratch so the overlay lands on any base.
FROM dhi.io/debian-base:trixie-dev AS build
ARG TARGETARCH
ARG GO_VERSION
RUN apt-get update \
 && apt-get install -y --no-install-recommends ca-certificates curl \
 && rm -rf /var/lib/apt/lists/*

RUN set -eux; \
    tarball="go${GO_VERSION}.linux-${TARGETARCH}.tar.gz"; \
    curl -fsSL -o "/tmp/${tarball}" "https://dl.google.com/go/${tarball}"; \
    sha256=$(curl -fsSL "https://dl.google.com/go/${tarball}.sha256"); \
    echo "${sha256}  /tmp/${tarball}" | sha256sum -c -; \
    mkdir -p /out/usr/local; \
    tar -xzf "/tmp/${tarball}" -C /out/usr/local; \
    /out/usr/local/go/bin/go version | grep -q "go${GO_VERSION} "; \
    chown -R 0:0 /out

FROM scratch
COPY --from=build /out /
ENTRYPOINT ["/usr/local/go/bin/go"]
CMD ["version"]
