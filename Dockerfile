# Builder stage
FROM docker.io/library/golang:1.25.3 AS builder
WORKDIR /build
# Copy go mod files and download dependencies
COPY ./src/go.mod ./src/go.sum ./
RUN go mod download
# Copy the rest of the source code
COPY ./src/ ./
# Build the helmizer binary with CGO disabled for a fully static binary
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o helmizer .

# Final stage
FROM scratch AS final
# Copy the helmizer binary to a known location
COPY --from=builder /build/helmizer /usr/local/bin/helmizer
# By default, set the binary as the entry point in case you want to run it
ENTRYPOINT ["/usr/local/bin/helmizer"]
