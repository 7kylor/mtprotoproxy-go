package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	// Protocol constants
	ObfuscatedTag = 0xdddddddd
	FakeTLSTag    = 0xeeeeeeee

	// Buffer sizes
	BufferSize = 4096

	// Telegram DCs (UAE optimized)
	TelegramDC5 = "91.108.56.130:443"   // Singapore (best for UAE)
	TelegramDC2 = "149.154.167.51:443"  // Amsterdam
	TelegramDC4 = "149.154.167.91:443"  // Amsterdam
	TelegramDC1 = "149.154.175.53:443"  // Miami
	TelegramDC3 = "149.154.175.100:443" // Miami
)

// Metrics
var (
	connectionsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mtproto_connections_total",
			Help: "Total number of connections",
		},
		[]string{"status"},
	)

	connectionsActive = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "mtproto_connections_active",
			Help: "Currently active connections",
		},
	)

	bytesTransferred = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mtproto_bytes_transferred_total",
			Help: "Total bytes transferred",
		},
		[]string{"direction", "datacenter"},
	)
)

func init() {
	prometheus.MustRegister(connectionsTotal)
	prometheus.MustRegister(connectionsActive)
	prometheus.MustRegister(bytesTransferred)
}

// Secret structure
type Secret struct {
	Key    []byte
	Type   byte
	Domain string
}

// Parse secret from hex string
func parseSecret(secretHex string) (*Secret, error) {
	secretBytes, err := hex.DecodeString(secretHex)
	if err != nil {
		return nil, err
	}

	if len(secretBytes) < 17 {
		return nil, fmt.Errorf("secret too short")
	}

	secret := &Secret{
		Type: secretBytes[0],
		Key:  secretBytes[1:17], // 16 bytes key
	}

	// Extract domain for FakeTLS
	if secret.Type == 0xee && len(secretBytes) > 17 {
		secret.Domain = string(secretBytes[17:])
	}

	return secret, nil
}

// Connection represents a client connection
type Connection struct {
	client   net.Conn
	telegram net.Conn
	secret   *Secret
	cipher   cipher.Stream
	decipher cipher.Stream
}

// Initialize encryption for obfuscated connections
func (c *Connection) initCrypto() error {
	if c.secret.Type != 0xdd && c.secret.Type != 0xee {
		return nil // No encryption for simple connections
	}

	// Create AES cipher
	block, err := aes.NewCipher(c.secret.Key)
	if err != nil {
		return err
	}

	// Generate random IV
	iv := make([]byte, aes.BlockSize)
	if _, err := rand.Read(iv); err != nil {
		return err
	}

	// Initialize CTR mode
	c.cipher = cipher.NewCTR(block, iv)
	c.decipher = cipher.NewCTR(block, iv)

	return nil
}

// Handle client connection
func (c *Connection) handleConnection() {
	defer c.client.Close()
	if c.telegram != nil {
		defer c.telegram.Close()
	}

	connectionsActive.Inc()
	defer connectionsActive.Dec()

	log.Printf("New connection from %s", c.client.RemoteAddr())

	// Initialize crypto if needed
	if err := c.initCrypto(); err != nil {
		log.Printf("Failed to initialize crypto: %v", err)
		connectionsTotal.WithLabelValues("crypto_error").Inc()
		return
	}

	// Read initial data from client
	buffer := make([]byte, BufferSize)
	n, err := c.client.Read(buffer)
	if err != nil {
		log.Printf("Failed to read from client: %v", err)
		connectionsTotal.WithLabelValues("read_error").Inc()
		return
	}

	// Handle different protocol types
	var telegramData []byte
	switch c.secret.Type {
	case 0xef: // Simple obfuscated
		telegramData = c.handleSimpleObfuscated(buffer[:n])
	case 0xdd: // Secured obfuscated
		telegramData = c.handleSecuredObfuscated(buffer[:n])
	case 0xee: // FakeTLS
		telegramData = c.handleFakeTLS(buffer[:n])
	default:
		log.Printf("Unknown secret type: %02x", c.secret.Type)
		connectionsTotal.WithLabelValues("unknown_protocol").Inc()
		return
	}

	if telegramData == nil {
		log.Printf("Failed to process protocol data")
		connectionsTotal.WithLabelValues("protocol_error").Inc()
		return
	}

	// Connect to Telegram
	if err := c.connectToTelegram(); err != nil {
		log.Printf("Failed to connect to Telegram: %v", err)
		connectionsTotal.WithLabelValues("telegram_error").Inc()
		return
	}

	// Send initial data to Telegram
	if _, err := c.telegram.Write(telegramData); err != nil {
		log.Printf("Failed to write to Telegram: %v", err)
		connectionsTotal.WithLabelValues("telegram_write_error").Inc()
		return
	}

	connectionsTotal.WithLabelValues("success").Inc()

	// Start bidirectional relay
	c.relay()
}

// Handle simple obfuscated protocol (0xef)
func (c *Connection) handleSimpleObfuscated(data []byte) []byte {
	if len(data) < 64 {
		return nil
	}

	// Simple obfuscated: remove first byte and return MTProto data
	return data[1:]
}

// Handle secured obfuscated protocol (0xdd)
func (c *Connection) handleSecuredObfuscated(data []byte) []byte {
	if len(data) < 64 {
		return nil
	}

	// Decrypt data using AES-CTR
	decrypted := make([]byte, len(data))
	c.decipher.XORKeyStream(decrypted, data)

	return decrypted
}

// Handle FakeTLS protocol (0xee)
func (c *Connection) handleFakeTLS(data []byte) []byte {
	if len(data) < 64 {
		return nil
	}

	// For FakeTLS, we need to extract the actual MTProto data
	// Skip TLS handshake simulation and extract MTProto payload

	// Look for MTProto marker after TLS headers
	for i := 0; i < len(data)-4; i++ {
		if binary.LittleEndian.Uint32(data[i:i+4]) == ObfuscatedTag {
			// Found MTProto data, decrypt if needed
			mtprotoData := data[i:]
			if c.decipher != nil {
				decrypted := make([]byte, len(mtprotoData))
				c.decipher.XORKeyStream(decrypted, mtprotoData)
				return decrypted
			}
			return mtprotoData
		}
	}

	// If no MTProto marker found, assume the whole payload is encrypted MTProto
	if c.decipher != nil {
		decrypted := make([]byte, len(data))
		c.decipher.XORKeyStream(decrypted, data)
		return decrypted
	}

	return data
}

// Connect to Telegram servers (UAE optimized)
func (c *Connection) connectToTelegram() error {
	// Try UAE-optimized DCs in order of preference
	datacenters := []string{
		TelegramDC5, // Singapore (best for UAE)
		TelegramDC2, // Amsterdam
		TelegramDC4, // Amsterdam
		TelegramDC1, // Miami
		TelegramDC3, // Miami
	}

	var lastErr error
	for _, dc := range datacenters {
		conn, err := net.DialTimeout("tcp", dc, 10*time.Second)
		if err != nil {
			lastErr = err
			continue
		}

		c.telegram = conn
		log.Printf("Connected to Telegram DC: %s", dc)
		return nil
	}

	return fmt.Errorf("failed to connect to any Telegram DC: %v", lastErr)
}

// Relay data between client and Telegram
func (c *Connection) relay() {
	var wg sync.WaitGroup
	wg.Add(2)

	// Client to Telegram
	go func() {
		defer wg.Done()
		c.relayData(c.client, c.telegram, "client_to_telegram")
	}()

	// Telegram to Client
	go func() {
		defer wg.Done()
		c.relayData(c.telegram, c.client, "telegram_to_client")
	}()

	wg.Wait()
}

// Relay data in one direction
func (c *Connection) relayData(src, dst net.Conn, direction string) {
	buffer := make([]byte, BufferSize)

	for {
		n, err := src.Read(buffer)
		if err != nil {
			if err != io.EOF {
				log.Printf("Read error (%s): %v", direction, err)
			}
			break
		}

		data := buffer[:n]

		// Apply encryption/decryption if needed
		if c.cipher != nil && direction == "telegram_to_client" {
			encrypted := make([]byte, n)
			c.cipher.XORKeyStream(encrypted, data)
			data = encrypted
		}

		_, err = dst.Write(data)
		if err != nil {
			log.Printf("Write error (%s): %v", direction, err)
			break
		}

		// Update metrics
		bytesTransferred.WithLabelValues(direction, "DC5_SIN").Add(float64(n))
	}
}

// MTProto Proxy
type MTProtoProxy struct {
	secret   *Secret
	listener net.Listener
}

// Create new proxy
func NewMTProtoProxy(secretHex string, bindAddr string) (*MTProtoProxy, error) {
	secret, err := parseSecret(secretHex)
	if err != nil {
		return nil, fmt.Errorf("invalid secret: %v", err)
	}

	listener, err := net.Listen("tcp", bindAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to bind: %v", err)
	}

	return &MTProtoProxy{
		secret:   secret,
		listener: listener,
	}, nil
}

// Start proxy
func (p *MTProtoProxy) Start() error {
	log.Printf("MTProto proxy listening on %s", p.listener.Addr())
	log.Printf("Secret type: %02x", p.secret.Type)
	if p.secret.Domain != "" {
		log.Printf("FakeTLS domain: %s", p.secret.Domain)
	}

	for {
		clientConn, err := p.listener.Accept()
		if err != nil {
			log.Printf("Accept error: %v", err)
			continue
		}

		// Handle connection in goroutine
		go func() {
			conn := &Connection{
				client: clientConn,
				secret: p.secret,
			}
			conn.handleConnection()
		}()
	}
}

// Generate secret
func generateSecret(domain string) string {
	// Generate 16 random bytes for key
	key := make([]byte, 16)
	rand.Read(key)

	// Create FakeTLS secret (0xee prefix)
	secret := append([]byte{0xee}, key...)
	secret = append(secret, []byte(domain)...)

	return hex.EncodeToString(secret)
}

// Auto-detect available port
func findAvailablePort(preferred int) int {
	for port := preferred; port <= preferred+100; port++ {
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			listener.Close()
			return port
		}
	}
	return preferred // fallback
}

// Get external IP
func getExternalIP() string {
	resp, err := http.Get("https://ipinfo.io/ip")
	if err != nil {
		return "YOUR_SERVER_IP"
	}
	defer resp.Body.Close()

	ip, err := io.ReadAll(resp.Body)
	if err != nil {
		return "YOUR_SERVER_IP"
	}

	return string(ip)
}

func main() {
	// Generate or use existing secret
	var secret string
	if len(os.Args) > 1 {
		secret = os.Args[1]
	} else {
		secret = generateSecret("google.com")
		log.Printf("Generated new secret: %s", secret)
	}

	// Find available port
	port := findAvailablePort(443)
	bindAddr := fmt.Sprintf("0.0.0.0:%d", port)

	// Create proxy
	proxy, err := NewMTProtoProxy(secret, bindAddr)
	if err != nil {
		log.Fatal(err)
	}

	// Start metrics server
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Printf("Metrics server starting on :8080")
		if err := http.ListenAndServe(":8080", nil); err != nil {
			log.Printf("Metrics server error: %v", err)
		}
	}()

	// Print connection info
	externalIP := getExternalIP()
	log.Printf("Starting MTProto proxy with:")
	log.Printf("- Protocol: MTProto 2.0 with obfuscation")
	log.Printf("- Secret type: %02x", proxy.secret.Type)
	if proxy.secret.Domain != "" {
		log.Printf("- FakeTLS domain: %s", proxy.secret.Domain)
	}
	log.Printf("- UAE-optimized routing (Singapore DC5 priority)")
	log.Printf("- Prometheus metrics on :8080/metrics")

	// Generate Telegram URL
	shortSecret := secret
	if len(shortSecret) > 34 {
		shortSecret = shortSecret[:34]
	}
	telegramURL := fmt.Sprintf("tg://proxy?server=%s&port=%d&secret=%s",
		externalIP, port, shortSecret)

	log.Printf("Telegram URL: %s", telegramURL)

	// Start proxy
	log.Fatal(proxy.Start())
}
