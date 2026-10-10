# Builder stage
FROM --platform=$BUILDPLATFORM docker.io/library/golang:1.26.7@sha256:e30143be198ab04cf7ba25fba83ab3a692ca584c994aad0bf131fa0eb32dd8c1 AS builder
ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
WORKDIR /build
# Copy go mod files and download dependencies
COPY ./src/go.mod ./src/go.sum ./
RUN go mod download
# Copy the rest of the source code
COPY ./src/ ./
# Build the helmizer binary with CGO disabled for a fully static binary
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} go build -ldflags="-s -w -X main.version=${VERSION}" -o helmizer .

# Final stage
FROM scratch AS final
# Copy the helmizer binary to a known location
COPY --from=builder /build/helmizer /usr/local/bin/helmizer
# Provide CA certificates for TLS
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
# By default, set the binary as the entry point in case you want to run it
USER 65532:65532
ENTRYPOINT ["/usr/local/bin/helmizer"]
