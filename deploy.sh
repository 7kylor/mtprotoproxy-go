#!/usr/bin/env bash
# deploy.sh — build and run the MTProto proxy container on a Linux server
# Usage: ./deploy.sh [bind_port] [image_tag]
#   bind_port  – Public TCP port to listen on (defaults to 443)
#   image_tag  – Optional docker tag name (defaults to mtproxy:latest)
set -euo pipefail

BIND_PORT=${1:-443}
IMAGE_TAG=${2:-mtproxy:latest}

# Function to check if a port is in use
port_in_use() {
    local port=$1
    if command -v lsof &>/dev/null; then
        lsof -i ":${port}" >/dev/null 2>&1
    elif command -v ss &>/dev/null; then
        ss -tuln | grep -q ":${port} "
    elif command -v netstat &>/dev/null; then
        netstat -tuln | grep -q ":${port} "
    else
        # Fallback: try to bind to the port
        (timeout 1 bash -c "exec 3<>/dev/tcp/localhost/${port}" 2>/dev/null && exec 3<&- && exec 3>&-) && return 0 || return 1
    fi
}

# Find an available port if the default is in use
find_available_port() {
    local start_port=$1
    local port=$start_port
    
    while port_in_use "$port"; do
        echo "Port $port is in use, trying $((port + 1))..." >&2
        port=$((port + 1))
        
        # Prevent infinite loop
        if [ $port -gt $((start_port + 100)) ]; then
            echo "Could not find available port in range $start_port-$((start_port + 100))" >&2
            exit 1
        fi
    done
    
    echo "$port"
}

# Check if requested port is available, find alternative if not
if port_in_use "$BIND_PORT"; then
    echo "Port $BIND_PORT is already in use."
    BIND_PORT=$(find_available_port "$BIND_PORT")
    echo "Using port $BIND_PORT instead."
fi

# Determine public IP for invite URL
if command -v curl &>/dev/null; then
  PUBLIC_IP=$(curl -s https://api.ipify.org || echo "$(hostname -I | awk '{print $1}' 2>/dev/null || echo '127.0.0.1')")
else
  PUBLIC_IP="$(hostname -I | awk '{print $1}' 2>/dev/null || echo '127.0.0.1')"
fi

echo "Using public IP: $PUBLIC_IP"
echo "Using port: $BIND_PORT"

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
  elif command -v brew &>/dev/null; then
    echo "On macOS, please install Docker Desktop from https://www.docker.com/products/docker-desktop/"
    exit 1
  else
    echo "Cannot install Docker automatically. Please install manually."
    exit 1
  fi
fi

# Skip group management on macOS (Docker Desktop doesn't need it)
if [[ "$OSTYPE" != "darwin"* ]]; then
  # Add current user to docker group if not already added
  if ! groups | grep -q docker; then
    echo "Adding user to docker group..."
    if command -v usermod &>/dev/null; then
      sudo usermod -aG docker "$USER"
      echo "Please log out and back in for group changes to take effect, then run this script again."
      exit 0
    else
      echo "Warning: Could not add user to docker group. You may need to run docker commands with sudo."
    fi
  fi
fi

echo "Building Docker image..."
docker build -t "$IMAGE_TAG" .

echo "Stopping existing mtproxy container if running..."
docker rm -f mtproxy 2>/dev/null || true

echo "Starting new mtproxy container..."
docker run -d \
  --name mtproxy \
  --restart unless-stopped \
  -p "${BIND_PORT}:3128" \
  -p 3129:3129 \
  -e MTG_BIND=0.0.0.0:3128 \
  -e ADVERTISED_HOST="$PUBLIC_IP" \
  "$IMAGE_TAG"

# Show connection URL
echo "Waiting for proxy to start..."
sleep 3

# Extract secret and URL from container logs
SECRET=$(docker logs mtproxy 2>&1 | grep -o "Generated secret: [a-fA-F0-9]*" | sed 's/Generated secret: //' | head -1)
if [ -z "$SECRET" ]; then
  SECRET=$(docker logs mtproxy 2>&1 | grep -o "secret=[a-fA-F0-9]*" | sed 's/secret=//' | head -1)
fi

if [ -n "$SECRET" ]; then
  URL="tg://proxy?server=${PUBLIC_IP}&port=${BIND_PORT}&secret=${SECRET}"
  echo -e "\ MTProto proxy is running!"
  echo -e "\ Telegram connection URL:"
  echo "$URL"
  echo -e "\ Prometheus metrics:"
  echo "http://${PUBLIC_IP}:3129/metrics"
  echo -e "\n🔧 Management commands:"
  echo "  docker logs mtproxy     # View logs"
  echo "  docker stop mtproxy     # Stop proxy"
  echo "  docker restart mtproxy  # Restart proxy"
else
  echo -e "\  Could not extract connection URL. Check logs with:"
  echo "docker logs mtproxy"
fi 