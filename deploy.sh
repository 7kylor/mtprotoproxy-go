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
        ss -ln | grep -q ":$port "
    elif command -v netstat >/dev/null 2>&1; then
        netstat -ln | grep -q ":$port "
    else
        return 1  # Cannot check, assume available
    fi
}

# Find available port starting from 443
BIND_PORT=443
while is_port_in_use $BIND_PORT; do
    echo -e "${YELLOW}Port $BIND_PORT is in use, trying $((BIND_PORT + 1))...${NC}"
    BIND_PORT=$((BIND_PORT + 1))
    if [ $BIND_PORT -gt 9999 ]; then
        echo -e "${RED}Could not find available port${NC}"
        exit 1
    fi
done

echo -e "${GREEN}Using port: $BIND_PORT${NC}"

# Build image
IMAGE_TAG="mtproxy:latest"
echo "Building Docker image..."
docker build -t "$IMAGE_TAG" .

# Stop existing container
echo "Stopping existing mtproxy container..."
docker stop mtproxy 2>/dev/null || true
docker rm mtproxy 2>/dev/null || true

echo "Starting new mtproxy container..."
docker run -d \
  --name mtproxy \
  --restart unless-stopped \
  -p "${BIND_PORT}:3128" \
  -e MTG_BIND=0.0.0.0:3128 \
  -e ADVERTISED_HOST="$PUBLIC_IP" \
  "nineseconds/mtg:2" \
  simple-run \
  --prefer-ip=prefer-ipv4 \
  --doh-ip=91.108.56.130 \
  0.0.0.0:3128 \
  $(docker run --rm "nineseconds/mtg:2" generate-secret google.com)

# Show connection URL
echo "Waiting for proxy to start..."
sleep 3

# Extract secret and URL from container logs  
SECRET=$(docker logs mtproxy 2>&1 | grep -o "secret=[a-fA-F0-9]*" | sed 's/secret=//' | head -1)
if [ -z "$SECRET" ]; then
  # Try to extract from generate-secret output
  SECRET=$(docker logs mtproxy 2>&1 | grep -o "[a-fA-F0-9]\{32,\}" | head -1)
fi

if [ -n "$SECRET" ]; then
  URL="tg://proxy?server=${PUBLIC_IP}&port=${BIND_PORT}&secret=${SECRET}"
  echo -e "${GREEN} MTProto proxy is running!${NC}"
  echo -e "${BLUE}Telegram connection URL:${NC}"
  echo "$URL"
  echo ""
  echo -e "${YELLOW} Management commands:${NC}"
  echo "  docker logs mtproxy     # View logs"
  echo "  docker stop mtproxy     # Stop proxy"
  echo "  docker restart mtproxy  # Restart proxy"
  echo ""
  echo -e "${BLUE}Singapore Datacenter Optimized:${NC}"
  echo "  DNS Server: 91.108.56.130 (Telegram DC5 Singapore)"
  echo "  IP Preference: IPv4 preferred for better routing"
  echo ""
else
  echo -e "${RED}Warning: Could not extract secret from container logs${NC}"
  echo "Use 'docker logs mtproxy' to check container status"
fi 