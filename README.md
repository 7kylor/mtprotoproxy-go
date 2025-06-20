# Complete MTProto Proxy - Production Ready Implementation

A comprehensive, high-performance MTProto proxy implementation built from scratch in Go, featuring all advanced MTProto 2.0 capabilities for production deployment in UAE and worldwide.

## Features

### **Complete MTProto 2.0 Implementation**

- **Full Protocol Support**: Complete MTProto 2.0 protocol implementation with proper message handling
- **All Transport Modes**: Abridged, Intermediate, Padded Intermediate, and Full transport protocols
- **Native Go Implementation**: Built from scratch without external MTProto dependencies

### **Advanced Security & Obfuscation**

- **Multiple Obfuscation Layers**:
  - Obfuscated2 protocol for deep packet inspection bypass
  - FakeTLS (SNI fronting) - traffic appears as HTTPS to `google.com`
  - ChaCha20 encryption for performance and security
- **Anti-Replay Protection**: SHA-256 based replay attack prevention with configurable TTL
- **Perfect Forward Secrecy**: Session-based key derivation

### **UAE-Optimized Datacenter Routing**

- **Intelligent DC Selection**: Prioritizes nearest datacenters for UAE users:
  1. **Singapore (DC5)** - Primary choice (lowest latency)
  2. **Amsterdam (DC2/DC4)** - Secondary choice
  3. **Miami (DC1/DC3)** - Fallback option
- **IPv6 Preference**: Prioritizes IPv6 connections for better performance
- **Automatic Failover**: Falls back to IPv4 if IPv6 unavailable

### **High Performance & Scalability**

- **Connection Multiplexing**: Efficient connection pooling per datacenter
- **Concurrent Processing**: Goroutine-based concurrent connection handling
- **Optimized Buffers**: 64KB buffers for high-throughput data transfer
- **Connection Reuse**: Smart connection pooling reduces establishment overhead
- **Non-blocking Architecture**: Handles thousands of concurrent connections

### **Production-Ready Observability**

- **Prometheus Metrics**:
  - Connection counts (total, active)
  - Data transfer statistics by datacenter
  - Connection duration histograms
  - Error counters by type
  - Per-datacenter connection distribution
- **Grafana Integration**: Ready-to-use dashboard configuration
- **Real-time Monitoring**: Live connection and performance metrics

## Quick Start

### 1. One-Command Deployment

```bash
git clone <repository-url>
cd mtprotoproxy-go
./deploy.sh
```

### 2. Docker Compose (with Grafana)

```bash
docker-compose up -d
```

### 3. Manual Build

```bash
go build -o mtproxy .
BIND_ADDR=:443 ADVERTISED_HOST=your.server.ip ./mtproxy
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `BIND_ADDR` | `:443` | Listen address and port |
| `ADVERTISED_HOST` | Auto-detected | Public IP for client URLs |
| `SECRET` | Auto-generated | 32-character hex secret |
| `SNI_DOMAIN` | `google.com` | FakeTLS SNI domain |

### Advanced Configuration

```bash
# Use custom secret (32 hex characters)
export SECRET="your32characterhexsecrethere"

# Change SNI domain for FakeTLS
export SNI_DOMAIN="cloudflare.com"

# Custom bind address
export BIND_ADDR="0.0.0.0:8443"
```

## Architecture

### Technical Implementation

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Telegram      │    │   MTProto Proxy  │    │   Telegram      │
│   Client        │◄──►│                  │◄──►│   Datacenter    │
│                 │    │                  │    │                 │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                              │
                              ▼
                       ┌──────────────────┐
                       │   Prometheus     │
                       │   Metrics        │
                       └──────────────────┘
```

### Key Components

1. **Transport Layer Detection**: Automatically detects and handles all MTProto transport types
2. **Obfuscation Engine**: Multi-layer encryption with ChaCha20 and FakeTLS
3. **Connection Pool Manager**: Efficient datacenter connection pooling
4. **Anti-Replay Cache**: Memory-efficient replay attack prevention
5. **Metrics Collection**: Comprehensive Prometheus instrumentation

### Datacenter Optimization for UAE

The proxy automatically selects the optimal Telegram datacenter based on geographic proximity:

| Datacenter | Location | IPv4 | IPv6 | Priority |
|------------|----------|------|------|----------|
| **DC5** | Singapore | `91.108.56.130` | `2001:b28:f23f:f005::a` | **1** (Best) |
| **DC2** | Amsterdam | `149.154.167.51` | `2001:67c:4e8:f002::a` | **2** |
| **DC4** | Amsterdam | `149.154.167.91` | `2001:67c:4e8:f004::a` | **2** |
| **DC1** | Miami | `149.154.175.53` | `2001:b28:f23d:f001::a` | **3** |
| **DC3** | Miami | `149.154.175.100` | `2001:b28:f23d:f003::a` | **3** |

## Security Features

### Obfuscation Methods

1. **Simple Obfuscation** (`0xef`): Basic transport obfuscation
2. **Secured Obfuscation** (`0xdd`): Enhanced obfuscation with additional padding
3. **FakeTLS** (`0xee`): Traffic disguised as HTTPS with SNI

### Anti-Replay Protection

- **SHA-256 Hashing**: Messages hashed for replay detection
- **Time-based TTL**: 5-minute replay window
- **Memory Efficient**: LRU-style cache with configurable size
- **Automatic Cleanup**: Background goroutine removes expired entries

### Encryption

- **ChaCha20**: Modern stream cipher for high performance
- **Session Keys**: Unique keys per connection
- **Perfect Forward Secrecy**: Keys derived from session data

## Monitoring & Observability

### Prometheus Metrics

Access metrics at `http://localhost:8080/metrics`:

```bash
# Active connections
mtproto_connections_active

# Total connections handled
mtproto_connections_total

# Data transfer by datacenter
mtproto_bytes_transferred_total{direction="client_to_telegram",datacenter="DC5_SIN"}

# Connection duration distribution
mtproto_connection_duration_seconds

# Error counts by type
mtproto_errors_total{type="replay_attack"}

# Per-datacenter connection counts
mtproto_datacenter_connections{datacenter="DC5",location="SIN"}
```

### Grafana Dashboard

The included Grafana setup provides:

- Real-time connection graphs
- Datacenter distribution charts
- Error rate monitoring
- Performance metrics

## Production Deployment

### System Requirements

- **RAM**: 512MB minimum, 2GB recommended
- **CPU**: 1 core minimum, 2+ cores recommended
- **Network**: Stable internet connection with low latency to Singapore/Amsterdam
- **Ports**: 443 (proxy), 8080 (metrics), optional 3000 (Grafana)

### Docker Production Setup

```yaml
version: '3.8'
services:
  mtproxy:
    image: mtproxy:latest
    ports:
      - "443:443"
      - "8080:8080"
    environment:
      - BIND_ADDR=0.0.0.0:443
      - ADVERTISED_HOST=your.domain.com
      - SNI_DOMAIN=google.com
    restart: unless-stopped
    deploy:
      resources:
        limits:
          memory: 1G
        reservations:
          memory: 512M
```

### Performance Tuning

```bash
# For high-traffic deployments
docker run -d \
  --name mtproxy \
  --restart unless-stopped \
  -p 443:443 \
  -p 8080:8080 \
  -e BIND_ADDR=0.0.0.0:443 \
  -e ADVERTISED_HOST=your.server.ip \
  --memory=2g \
  --cpus=2 \
  mtproxy:latest
```

## Advanced Usage

### Custom Secret Generation

```bash
# Generate FakeTLS secret for custom domain
SECRET="ee$(openssl rand -hex 16)$(echo -n "yourdomain.com" | xxd -p)"
```

### Load Balancing

Deploy multiple instances with the same secret:

```bash
# Instance 1
docker run -d --name mtproxy1 -p 443:443 -e SECRET=$SECRET mtproxy

# Instance 2  
docker run -d --name mtproxy2 -p 444:443 -e SECRET=$SECRET mtproxy
```

### Monitoring Integration

```bash
# Export metrics to external monitoring
curl -s http://localhost:8080/metrics | \
  prometheus-remote-write --url=https://your-prometheus-server/api/v1/write
```

## Troubleshooting

### Common Issues

1. **Port 443 in use**: Script automatically finds next available port
2. **Connection timeouts**: Check datacenter connectivity and IPv6 support
3. **High memory usage**: Tune anti-replay cache size in configuration

### Debug Commands

```bash
# View live logs
docker logs -f mtproxy

# Check connection stats
curl -s http://localhost:8080/metrics | grep mtproto_connections

# Test connectivity to datacenters
for dc in 91.108.56.130 149.154.167.51; do
  echo "Testing $dc..."
  timeout 5 nc -v $dc 443
done
```

### Performance Optimization

```bash
# Increase connection limits (Linux)
echo 'net.core.somaxconn = 65535' >> /etc/sysctl.conf
echo 'net.ipv4.tcp_max_syn_backlog = 65535' >> /etc/sysctl.conf
sysctl -p
```

## License

MIT License - See LICENSE file for details.

## Contributing

Contributions welcome! Please read our contributing guidelines and submit pull requests for any improvements.

## Security

For security issues, please email <security@yourproject.com> instead of using the issue tracker.

---

**Built with ❤️ for the Telegram community**

*This implementation provides enterprise-grade MTProto proxy functionality with production-ready features for high-performance, secure Telegram access.*
