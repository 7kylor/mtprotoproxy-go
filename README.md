# mtprotoproxy-go

Fast, minimal MTProto proxy powered by [mtg](https://github.com/9seconds/mtg) written in Go.

* FakeTLS with one secret (auto-generated if you don't provide one)
* Anti-replay cache and IPv6-preferred networking
* Prometheus metrics on `/metrics` (port 3129)
* Built as a static binary (~6 MB) and shipped in a distroless container

---

## Quick start (local)

```bash
git clone https://github.com/yourname/mtprotoproxy-go.git
cd mtprotoproxy-go
# build and run in foreground
docker compose up --build
```

The proxy listens on **127.0.0.1:3128** and prints a ready-to-use **tg://proxy** link.

---

## One-liner deployment on a VPS

```
scp -r . vps:/opt/mtproxy
ssh vps
cd /opt/mtproxy && ./deploy.sh 443   # optional: <public_port> [image_tag]
```

* Installs Docker if missing
* Builds the multi-arch image
* Starts container (`--restart unless-stopped`)
* Prints the Telegram client URL and the Prometheus endpoint

---

## Environment variables (all optional)

| Name             | Default      | Description                              |
|------------------|--------------|------------------------------------------|
| `MTG_SECRET`     | auto-gen     | Proxy secret (hex or base64)             |
| `MTG_BIND`       | `:3128`      | Internal listen address                  |
| `MTG_PREFER_IP`  | `prefer-ipv6`| How to prefer IPv4/IPv6 when dialing DCs |
| `ADVERTISED_HOST`| detected IP  | Host/IP used in the printed invite URL   |

---

## Building without Docker

```bash
go 1.22+
go mod tidy
go build -o mtproxy .
./mtproxy                # runs on :3128
```

---

## Metrics

Prometheus exporter is always enabled on **:3129/metrics** inside the container.

---

## License

MIT – same as mtg.
