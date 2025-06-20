package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/9seconds/mtg/v2/antireplay"
	"github.com/9seconds/mtg/v2/events"
	"github.com/9seconds/mtg/v2/ipblocklist/empty"
	"github.com/9seconds/mtg/v2/logger/std"
	"github.com/9seconds/mtg/v2/mtglib"
	"github.com/9seconds/mtg/v2/network/direct"
	"github.com/9seconds/mtg/v2/stats/prometheus"
)

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	bind := getenv("MTG_BIND", ":3128")

	secretStr := os.Getenv("MTG_SECRET")
	if secretStr == "" {
		host := getenv("MTG_SNI", "google.com")
		secret := mtglib.GenerateSecret(host)
		secretStr = secret.Hex()
		// persist into env for downstream tools in same process
		_ = os.Setenv("MTG_SECRET", secretStr)
	}

	prefIP := os.Getenv("MTG_PREFER_IP") // "" -> default "prefer-ipv6"
	cacheMBEnv := os.Getenv("MTG_ANTIREPLAY_MB")
	cacheMB := 64 // default 64 MiB if env missing or invalid
	if cacheMBEnv != "" {
		if v, err := strconv.Atoi(cacheMBEnv); err == nil && v > 0 {
			cacheMB = v
		}
	}

	secret, err := mtglib.ParseSecret(secretStr)
	if err != nil {
		log.Fatalf("invalid secret: %v", err)
	}

	anti := antireplay.NewCache(uint(cacheMB) << 20) // default 64 MB
	network := direct.New(prefIP)
	stream := events.NewStream(1024)
	logBackend := std.New()

	proxy, err := mtglib.NewProxy(mtglib.ProxyOpts{
		Secret:                   secret,
		Network:                  network,
		AntiReplayCache:          anti,
		IPBlocklist:              empty.New(),
		EventStream:              stream,
		Logger:                   logBackend,
		Concurrency:              8192,
		DomainFrontingPort:       443,
		AllowFallbackOnUnknownDC: true,
	})
	if err != nil {
		log.Fatalf("proxy init: %v", err)
	}

	// Prometheus exporter
	go prometheus.Export(stream, ":3129", "mtg")

	ln, err := net.Listen("tcp", bind)
	if err != nil {
		log.Fatalf("listen %s: %v", bind, err)
	}
	log.Printf("proxy listening on %s (FakeTLS SNI=%s)", bind, secret.Host)

	// Construct and print client URL for convenience
	advertisedHost := getenv("ADVERTISED_HOST", "127.0.0.1")
	_, port, _ := net.SplitHostPort(bind)
	inviteURL := fmt.Sprintf("tg://proxy?server=%s&port=%s&secret=%s", advertisedHost, port, secret.Hex())
	log.Printf("Telegram client URL: %s", inviteURL)

	// graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		log.Println("shutting down …")
		_ = ln.Close()
		proxy.Shutdown()
	}()

	if err := proxy.Serve(ln); err != nil {
		if ctx.Err() == nil { // not cancelled
			log.Fatalf("proxy serve: %v", err)
		}
	}
	time.Sleep(time.Second) // let goroutines flush
}
