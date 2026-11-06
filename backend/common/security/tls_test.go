// path: backend/common/security/tls_test.go
package security

import (
	"crypto/tls"
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"
)

func findCertsDir() string {
	candidates := []string{
		"../../../certs",
		"../../certs",
		"../certs",
		"certs",
	}
	for _, c := range candidates {
		if _, err := os.Stat(filepath.Join(c, "ca.crt")); err == nil {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}
	abs, _ := filepath.Abs("../../../certs")
	return abs
}

func TestLoadTLSConfigs(t *testing.T) {
	certsDir := findCertsDir()

	caPath := filepath.Join(certsDir, "ca.crt")
	serverCert := filepath.Join(certsDir, "user.crt")
	serverKey := filepath.Join(certsDir, "user.key")
	clientCert := filepath.Join(certsDir, "gateway.crt")
	clientKey := filepath.Join(certsDir, "gateway.key")

	// 1. Server TLS Config
	srvCfg, err := LoadServerTLSConfig(caPath, serverCert, serverKey)
	if err != nil {
		t.Fatalf("LoadServerTLSConfig failed: %v", err)
	}

	if srvCfg.ClientAuth != tls.RequireAndVerifyClientCert {
		t.Errorf("expected RequireAndVerifyClientCert, got %v", srvCfg.ClientAuth)
	}
	if srvCfg.MinVersion != tls.VersionTLS13 {
		t.Errorf("expected TLS 1.3 min version, got %v", srvCfg.MinVersion)
	}

	// 2. Client TLS Config
	cliCfg, err := LoadClientTLSConfig(caPath, clientCert, clientKey, "user")
	if err != nil {
		t.Fatalf("LoadClientTLSConfig failed: %v", err)
	}

	if cliCfg.ServerName != "user" {
		t.Errorf("expected ServerName 'user', got %s", cliCfg.ServerName)
	}
	if cliCfg.MinVersion != tls.VersionTLS13 {
		t.Errorf("expected TLS 1.3 min version, got %v", cliCfg.MinVersion)
	}

	// 3. Negative test: Missing file
	_, err = LoadServerTLSConfig("non-existent-ca.crt", serverCert, serverKey)
	if err == nil {
		t.Errorf("expected error for non-existent CA path, got nil")
	}
}

func TestMutualTLS_Handshake(t *testing.T) {
	certsDir := findCertsDir()

	caPath := filepath.Join(certsDir, "ca.crt")
	serverCert := filepath.Join(certsDir, "order.crt")
	serverKey := filepath.Join(certsDir, "order.key")
	clientCert := filepath.Join(certsDir, "gateway.crt")
	clientKey := filepath.Join(certsDir, "gateway.key")

	srvCfg, err := LoadServerTLSConfig(caPath, serverCert, serverKey)
	if err != nil {
		t.Fatalf("server TLS config error: %v", err)
	}

	// Start mTLS TCP Listener on loopback
	listener, err := tls.Listen("tcp", "127.0.0.1:0", srvCfg)
	if err != nil {
		t.Fatalf("tls.Listen error: %v", err)
	}
	defer listener.Close()

	addr := listener.Addr().String()

	// Server worker goroutine
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				buf := make([]byte, 16)
				n, _ := c.Read(buf)
				if n > 0 {
					_, _ = c.Write([]byte("PONG"))
				}
			}(conn)
		}
	}()

	// 1. Success case: Client presents valid mTLS certificate
	cliCfg, err := LoadClientTLSConfig(caPath, clientCert, clientKey, "localhost")
	if err != nil {
		t.Fatalf("client TLS config error: %v", err)
	}

	conn, err := tls.Dial("tcp", addr, cliCfg)
	if err != nil {
		t.Fatalf("mTLS dial failed with valid client cert: %v", err)
	}
	_, err = conn.Write([]byte("PING"))
	if err != nil {
		t.Fatalf("write error: %v", err)
	}
	resp := make([]byte, 4)
	_, err = io.ReadFull(conn, resp)
	if err != nil || string(resp) != "PONG" {
		t.Fatalf("expected PONG response over mTLS, got %s (err: %v)", string(resp), err)
	}
	conn.Close()

	// 2. Failure case: Insecure client connects without presenting client certificate
	insecureCliCfg := &tls.Config{
		InsecureSkipVerify: true,
	}
	rejectedConn, err := tls.Dial("tcp", addr, insecureCliCfg)
	if err == nil {
		// Attempting to read/write should trigger handshake termination
		buf := make([]byte, 10)
		_, readErr := rejectedConn.Read(buf)
		if readErr == nil {
			t.Fatalf("expected handshake failure when client certificate is missing, but connection succeeded")
		}
		rejectedConn.Close()
	}
}

