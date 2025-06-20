#!/usr/bin/env bash
# deploy.sh — build and run the MTProto proxy container on a Linux server
# Usage: ./deploy.sh <secret> [bind_port] [image_tag]
#   secret     – MTProto secret in hex or base64 form (required)
#   bind_port  – Public TCP port to listen on (defaults to 443)
#   image_tag  – Optional docker tag name (defaults to mtproxy:latest)
set -euo pipefail

BIND_PORT=${1:-443}
IMAGE_TAG=${2:-mtproxy:latest}

# Determine public IP for invite URL
if command -v curl &>/dev/null; then
  PUBLIC_IP=$(curl -s https://api.ipify.org || true)
else
  PUBLIC_IP="$(hostname -I | awk '{print $1}')"
fi

# Ensure Docker is available
if ! command -v docker &>/dev/null; then
  echo "Docker not found — attempting to install..."
  if command -v apt-get &>/dev/null; then
    sudo apt-get update -y
    sudo apt-get install -y docker.io
    sudo systemctl enable --now docker
  elif command -v yum &>/dev/null; then
    sudo yum install -y docker
    sudo systemctl enable --now docker
  else
    echo "Please install Docker manually for your distribution" >&2
    exit 1
  fi
fi

# Build the image (multi-arch friendly)
DOCKER_BUILDKIT=1 docker build --platform=linux/amd64 -t "$IMAGE_TAG" .

# Stop old container if running
if docker ps -a --format '{{.Names}}' | grep -q '^mtproxy$'; then
  docker rm -f mtproxy
fi

# Run the container
docker run -d \
  --name mtproxy \
  --restart unless-stopped \
  -p "${BIND_PORT}:3128" \
  -p 3129:3129 \
  -e MTG_BIND=0.0.0.0:3128 \
  -e ADVERTISED_HOST="$PUBLIC_IP" \
  "$IMAGE_TAG"

# Show connection URL
sleep 2
URL=$(docker logs mtproxy 2>&1 | grep -m1 "Telegram client URL" | sed -E 's/.*URL: //')

echo -e "\nMTProto proxy is up. Connect using:\n${URL}\nPrometheus metrics: http://${PUBLIC_IP}:3129/metrics" 