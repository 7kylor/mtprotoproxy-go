#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE} MTProto Proxy Deployment Script${NC}"
echo "=================================="

# Get public IP
echo -n "Detecting public IP... "
PUBLIC_IP=$(curl -s --max-time 10 https://api.ipify.org || echo "")
if [ -z "$PUBLIC_IP" ]; then
    echo -e "${RED}Failed${NC}"
    echo "Could not detect public IP. Please set it manually:"
    read -p "Enter your server's public IP: " PUBLIC_IP
else
    echo -e "${GREEN}$PUBLIC_IP${NC}"
fi

# Function to check if port is in use
is_port_in_use() {
    local port=$1
    if command -v lsof >/dev/null 2>&1; then
        lsof -i :$port >/dev/null 2>&1
    elif command -v ss >/dev/null 2>&1; then
        ss -tuln | grep ":$port " >/dev/null 2>&1
    elif command -v netstat >/dev/null 2>&1; then
        netstat -tuln | grep ":$port " >/dev/null 2>&1
    else
        # Fallback: try to bind to the port
        if timeout 1 bash -c "</dev/tcp/localhost/$port" 2>/dev/null; then
            return 0  # Port is in use
        else
            return 1  # Port is free
        fi
    fi
}

# Find available port starting from 443
BIND_PORT=443
while is_port_in_use $BIND_PORT; do
    echo -e "${YELLOW}Port $BIND_PORT is in use, trying $((BIND_PORT + 1))${NC}"
    BIND_PORT=$((BIND_PORT + 1))
    if [ $BIND_PORT -gt 65535 ]; then
        echo -e "${RED}No available ports found${NC}"
        exit 1
    fi
done

echo -e "${GREEN}Using port: $BIND_PORT${NC}"

# Set image tag
IMAGE_TAG="mtproxy:latest"

# Stop existing container if running
if docker ps -q -f name=mtproxy >/dev/null 2>&1; then
    echo "Stopping existing mtproxy container..."
    docker stop mtproxy >/dev/null 2>&1 || true
    docker rm mtproxy >/dev/null 2>&1 || true
fi

# Build the image
echo "Building MTProto proxy image..."
docker build -t "$IMAGE_TAG" .

if [ $? -ne 0 ]; then
    echo -e "${RED}Failed to build Docker image${NC}"
    exit 1
fi

echo -e "${GREEN}Image built successfully${NC}"

# Generate or use existing secret
SECRET=${SECRET:-""}

echo "Starting new mtproxy container..."
docker run -d \
  --name mtproxy \
  --restart unless-stopped \
  -p "${BIND_PORT}:443" \
  -p "8080:8080" \
  -e BIND_ADDR=0.0.0.0:443 \
  -e ADVERTISED_HOST="$PUBLIC_IP" \
  -e SECRET="$SECRET" \
  -e SNI_DOMAIN="${SNI_DOMAIN:-google.com}" \
  "$IMAGE_TAG"

# Show connection URL
echo "Waiting for proxy to start..."
sleep 5

# Extract secret and URL from container logs
SECRET=$(docker logs mtproxy 2>&1 | grep -o "Generated new secret: [a-fA-F0-9]*" | sed 's/Generated new secret: //' | head -1)
if [ -z "$SECRET" ]; then
  SECRET=$(docker logs mtproxy 2>&1 | grep -o "Secret: [a-fA-F0-9]*" | sed 's/Secret: //' | head -1)
fi

if [ -n "$SECRET" ]; then
  URL="tg://proxy?server=${PUBLIC_IP}&port=${BIND_PORT}&secret=${SECRET}"
  echo -e "${GREEN} MTProto proxy is running!${NC}"
  echo -e "${BLUE}Telegram connection URL:${NC}"
  echo "$URL"
  echo ""
  echo -e "${YELLOW} Management commands:${NC}"
  echo "  docker logs mtproxy         # View logs"
  echo "  docker stop mtproxy         # Stop proxy"
  echo "  curl http://localhost:8080/metrics  # View metrics"
  echo ""
  echo -e "${BLUE} Advanced Features Active:${NC}"
  echo "  - Full MTProto 2.0 protocol implementation"
  echo "  - Obfuscated2 and FakeTLS support"
  echo "  - Anti-replay protection"
  echo "  - UAE-optimized datacenter routing (prioritizes Singapore DC5)"
  echo "  - Connection multiplexing with pooling"
  echo "  - Prometheus metrics on port 8080"
else
  echo -e "${RED}Failed to extract connection details${NC}"
  echo "Check container logs with: docker logs mtproxy"
fi 