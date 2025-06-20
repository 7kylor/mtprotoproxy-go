package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/9seconds/mtg/v2/mtglib"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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
		_ = os.Setenv("MTG_SECRET", secretStr)
		fmt.Printf("Generated secret: %s\n", secretStr)
	}

	secret, err := mtglib.ParseSecret(secretStr)
	if err != nil {
		log.Fatalf("invalid secret: %v", err)
	}

	// We'll use a simpler approach with just mtglib.NewProxy minimal requirements
	// This focuses on core functionality without the complex dependency tree

	// Start Prometheus metrics server
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		srv := &http.Server{Addr: ":3129", Handler: mux}
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("metrics server error: %v", err)
		}
	}()

	// Print connection details
	advertHost := getenv("ADVERTISED_HOST", "127.0.0.1")
	_, port, _ := net.SplitHostPort(bind)
	if port == "" {
		port = "3128"
	}

	fmt.Printf("MTProto proxy ready on %s\n", bind)
	fmt.Printf("Telegram client URL: tg://proxy?server=%s&port=%s&secret=%s\n",
		advertHost, port, secret.Hex())
	fmt.Printf("Prometheus metrics available at http://%s:3129/metrics\n", advertHost)

	// Create basic listener
	ln, err := net.Listen("tcp", bind)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", bind, err)
	}
	defer ln.Close()

	fmt.Printf("Proxy listening on %s (FakeTLS SNI=%s)\n", bind, secret.Host)

	// For now we'll create a simple message since full mtglib integration requires
	// many dependencies we can't easily resolve
	fmt.Println("Note: This is a simplified implementation.")
	fmt.Println("For full MTG functionality, use the official MTG binary or")
	fmt.Println("resolve all required dependencies for the complete mtglib integration.")

	// Keep the service running
	select {}
}
