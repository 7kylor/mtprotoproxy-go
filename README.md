# mtprotoproxy-go

Fast, minimal MTProto proxy implementation in Go with auto-generated secrets and observability.

## Features

* **Auto-generated secrets** - No manual secret management required
* **FakeTLS with SNI fronting** - Uses google.com as default SNI hostname  
* **Prometheus metrics** - Built-in `/metrics` endpoint on port 3129
* **Minimal Docker image** - ~6 MB distroless container
* **Zero-config deployment** - Runs out of the box with sensible defaults

---

## Quick Start (Local Development)

```bash
# Clone and build
git clone <your-repo-url>
cd mtprotoproxy-go
go mod tidy
go build -o mtproxy .

# Run locally
./mtproxy
```

The proxy will:

* Auto-generate a secret with SNI=google.com
* Listen on `:3128` for MTProto connections  
* Serve Prometheus metrics on `:3129/metrics`
* Print a ready-to-use `tg://proxy` URL

---

## One-Touch VPS Deployment

Upload this project to your VPS and run:

```bash
# Make executable and deploy
chmod +x deploy.sh
./deploy.sh [public_port] [image_tag]

# Examples:
./deploy.sh                    # Uses port 443, image mtproxy:latest
./deploy.sh 8443              # Custom port
./deploy.sh 8443 mtproxy:v1   # Custom port and tag
```

The script will:

1. Auto-detect your public IP
2. Install Docker (if missing)  
3. Build the image
4. Run the container with auto-restart
5. Extract and display the Telegram connection URL

---

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `MTG_SECRET` | auto-generated | MTProto secret (hex format) |
| `MTG_BIND` | `:3128` | Listen address for proxy |
| `MTG_SNI` | `google.com` | SNI hostname for FakeTLS |
| `ADVERTISED_HOST` | auto-detected | Public IP for connection URLs |

---

## Docker Usage

```bash
# Build image
docker build -t mtproxy .

# Run with custom settings
docker run -d \
  --name mtproxy \
  --restart unless-stopped \
  -p 443:3128 \
  -p 3129:3129 \
  -e MTG_SNI=your-domain.com \
  -e ADVERTISED_HOST=YOUR_PUBLIC_IP \
  mtproxy

# View connection details
docker logs mtproxy
```

---

## Manual Build

If you prefer building without Docker:

```bash
# Install Go 1.22+
go mod tidy
CGO_ENABLED=0 go build -ldflags="-s -w" -o mtproxy .

# Run with custom config
MTG_BIND=:8443 MTG_SNI=example.com ./mtproxy
```

---

## Monitoring

* **Metrics**: `http://your-server:3129/metrics`
* **Container logs**: `docker logs mtproxy`
* **Management**: `docker {stop|restart|rm} mtproxy`

---

## Implementation Notes

This is a **simplified proxy implementation** that focuses on:

* Quick deployment and auto-configuration
* Basic MTProto secret generation and URL formatting
* Container orchestration and metrics exposure

For full MTG feature parity (complete MTProto protocol handling, advanced anti-replay protection, domain fronting, etc.), consider using the [official MTG binary](https://github.com/9seconds/mtg) or integrating the complete `mtglib` dependency tree.

---

## License

MIT License - Use freely for personal and commercial projects.
