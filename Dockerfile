# Use the offical golang image to create a binary.
# This is based on Debian and sets the GOPATH to /go.
# https://hub.docker.com/_/golang
FROM golang:1.23-bookworm AS builder

ARG APP_VERSION
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
RUN go build  -ldflags="-X 'main.Version=$APP_VERSION'" -v -o server ./cmd/
#RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
#    -ldflags="-w -s" \
#    -o /app/server \
#    cmd/

# Use the official Debian slim image for a lean production container.
# https://hub.docker.com/_/debian
# https://docs.docker.com/develop/develop-images/multistage-build/#use-multi-stage-builds
FROM debian:bookworm-slim

ARG TALOS_VERSION=v1.8.0
ARG K8S_VERSION=1.30.3
ARG YQ_VERSION=v4.44.5

RUN set -x && apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y \
    ca-certificates curl jq && \
    rm -rf /var/lib/apt/lists/* && \
    groupadd -r cockpit && \
    useradd -r -g cockpit -m cockpit



RUN curl -L https://github.com/siderolabs/talos/releases/download/${TALOS_VERSION}/talosctl-linux-amd64 -o /usr/local/bin/talosctl && chmod +x /usr/local/bin/talosctl
RUN curl -L https://dl.k8s.io/release/${K8S_VERSION}/bin/linux/amd64/kubectl -o /usr/local/bin/kubectl -o /usr/local/bin/ && chmod +x /usr/local/bin/kubectl
RUN curl -L https://github.com/mikefarah/yq/releases/download/${YQ_VERSION}/yq_linux_amd64 -o /usr/local/bin/yq -o /usr/local/bin/ && chmod +x /usr/local/bin/yq


# Copy the binary to the production image from the builder stage.
COPY --from=builder /app/server /app/server

# Copy local static to the container image.
COPY ./static /app/static
COPY ./templates /app/templates

# Set permissions
RUN chown -R cockpit:cockpit /app

# Switch to non-root user
USER cockpit

# Run the web service on container startup.
CMD ["/app/server"]