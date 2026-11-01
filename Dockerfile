# Builder stage
FROM --platform=$BUILDPLATFORM docker.io/library/golang:1.27.3@sha256:2920bb124f1a2e12a1d39f0cab756ae0068441ee1da7655dec7f5e374944095b AS builder
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
