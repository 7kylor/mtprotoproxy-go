# MTProto Proxy - Go Implementation

A lightweight, fast, and secure MTProto proxy for Telegram using the official MTG binary. This implementation provides FakeTLS (SNI fronting), automatic secret generation, and one-click deployment.

## Features

- **Official MTG Binary**: Uses the battle-tested MTG v2 binary for maximum compatibility
- **FakeTLS Support**: Traffic appears as regular HTTPS traffic to google.com (configurable)
- **Singapore Datacenter Optimized**: Configured to use Singapore (DC5) for optimal Asia-Pacific routing
- **Auto-Configuration**: Automatic secret generation and port detection  
- **Lightweight**: ~6MB Docker image using distroless base
- **Production Ready**: Includes restart policies and proper security settings
- **One-Click Deployment**: Automated setup script for VPS deployment

## Quick Start

### 1. Clone and Deploy

```bash
git clone <repository-url>
cd mtprotoproxy-go
./deploy.sh
```

The deployment script will:

- Auto-detect your public IP
- Find an available port (starting from 443)
- Build the Docker image
- Generate a secure secret with FakeTLS
- Start the proxy with restart policies
- Display the Telegram connection URL

### 2. Use the Connection URL

Copy the generated URL and share it with Telegram clients:

```
tg://proxy?server=YOUR_IP&port=PORT&secret=SECRET
```

## Manual Deployment

### Build Image

```bash
docker build -t mtproxy .
```

### Generate Secret

```bash
docker run --rm mtproxy generate-secret google.com
```

### Run Proxy

```bash
docker run -d \
  --name mtproxy \
  --restart unless-stopped \
  -p 443:3128 \
  mtproxy simple-run 0.0.0.0:3128 YOUR_SECRET
```

## Configuration

### Singapore Datacenter Optimization

The proxy is configured to optimize routing through Telegram's Singapore datacenter (DC5):

- **DNS Server**: Uses `91.108.56.130` (Telegram DC5 Singapore IP)
- **IP Preference**: Configured for IPv4 preference for better routing
- **Geographic Advantage**: Ideal for UAE, Asia-Pacific, and Middle East users

This configuration provides:

- Lower latency for users in the Asia-Pacific region
- Better routing through Telegram's Singapore infrastructure  
- Optimal performance for users in UAE, India, Southeast Asia, and nearby regions

### Environment Variables

You can customize the proxy behavior using these environment variables:

```bash
MTG_BIND=0.0.0.0:3128          # Bind address and port
ADVERTISED_HOST=your-ip        # Public IP for client connections
```

## Manual Configuration

### SNI Domain

By default, the proxy uses `google.com` for FakeTLS. You can generate secrets for other domains:

```bash
docker run --rm mtproxy generate-secret example.com
```

Choose domains that are:

- Commonly accessed from your server's location
- Not blocked in your region  
- Relevant to your hosting provider (e.g., digitalocean.com for DigitalOcean VPS)

## Management

### View Logs

```bash
docker logs mtproxy
```

### Restart Proxy

```bash
docker restart mtproxy
```

### Stop Proxy

```bash
docker stop mtproxy
```

### Update Proxy

```bash
./deploy.sh  # Rebuilds and redeploys
```

## Security Notes

1. **Firewall**: Consider using a firewall whitelist for additional security
2. **Port Selection**: The script automatically finds available ports if 443 is in use
3. **Secret Rotation**: Regularly regenerate secrets for enhanced security
4. **Domain Fronting**: Choose SNI domains carefully for your region

## Architecture

This implementation:

- Uses the official MTG v2 binary (most stable and feature-complete)
- Runs in a minimal distroless container for security
- Supports all MTG features including FakeTLS and anti-replay protection
- Provides automated deployment and management
- Optimized for Singapore datacenter routing

## Why MTG?

MTG (MTProto Go) is the recommended implementation because:

- **Battle-tested**: Used by thousands of users worldwide
- **Feature-complete**: Supports all modern MTProto features
- **Actively maintained**: Regular updates and security fixes
- **Performance**: Highly optimized Go implementation
- **Security**: Built-in anti-replay and domain fronting

## Troubleshooting

### Container Won't Start

```bash
docker logs mtproxy
```

### Port Already in Use

The deployment script automatically finds available ports. If you need a specific port:

```bash
# Stop other services using the port
sudo lsof -i :443
```

### Connection Issues

1. Verify your firewall allows the port
2. Check if your VPS provider blocks proxy traffic
3. Try generating a new secret with a different SNI domain

## Performance

- **CPU**: Minimal usage (~1% on average VPS)
- **Memory**: ~10-20MB RAM usage
- **Network**: No bandwidth overhead (direct relay)
- **Latency**: Minimal additional latency (<10ms typically)

## License

This project uses the official MTG binary, which is licensed under MIT License.
