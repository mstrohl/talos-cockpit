# Use the offical golang image to create a binary.
# This is based on Debian and sets the GOPATH to /go.
# https://hub.docker.com/_/golang
FROM golang:1.23-bookworm AS builder

# Create and change to the app directory.
WORKDIR /app

# Retrieve application dependencies.
# This allows the container build to reuse cached dependencies.
# Expecting to copy go.mod and if present go.sum.
COPY go.* ./
RUN go mod download

# Copy local code to the container image.
COPY . ./

# Build the binary.
RUN go build -v -o server

# Use the official Debian slim image for a lean production container.
# https://hub.docker.com/_/debian
# https://docs.docker.com/develop/develop-images/multistage-build/#use-multi-stage-builds
FROM debian:bookworm-slim

ARG TALOS_VERSION=v1.8.0
ARG K8S_VERSION=1.30.3

RUN set -x && apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y \
    ca-certificates curl jq && \
    rm -rf /var/lib/apt/lists/*

RUN curl -L https://github.com/siderolabs/talos/releases/download/${TALOS_VERSION}/talosctl-linux-amd64 -o /usr/local/bin/talosctl && chmod +x /usr/local/bin/talosctl
RUN curl -L https://dl.k8s.io/release/${K8S_VERSION}/bin/linux/amd64/kubectl -o /usr/local/bin/kubectl && chmod +x /usr/local/bin/kubectl


# Copy the binary to the production image from the builder stage.
COPY --from=builder /app/server /app/server

# Run the web service on container startup.
CMD ["/app/server"]