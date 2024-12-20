#!/bin/bash

# Variables
APP_NAME="talos-cockpit"
VERSION="v0.0.1"
REGISTRY="mstrohl"

# Build Docker image
docker build -t ${REGISTRY}/${APP_NAME}:${VERSION} .

# Optional: Push to registry
docker push ${REGISTRY}/${APP_NAME}:${VERSION}