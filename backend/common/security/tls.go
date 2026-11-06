// path: backend/common/security/tls.go
package security

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"google.golang.org/grpc/credentials"
)

// LoadServerTLSConfig builds a TLS config enforcing mutual TLS (mTLS).
// It requires and verifies client certificates against the provided CA pool.
func LoadServerTLSConfig(caCertPath, serverCertPath, serverKeyPath string) (*tls.Config, error) {
	caCertBytes, err := os.ReadFile(caCertPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate from %s: %w", caCertPath, err)
	}

	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caCertBytes) {
		return nil, fmt.Errorf("failed to parse CA certificate into cert pool")
	}

	serverCert, err := tls.LoadX509KeyPair(serverCertPath, serverKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load server keypair (%s, %s): %w", serverCertPath, serverKeyPath, err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    caPool,
		MinVersion:   tls.VersionTLS13,
	}, nil
}

// LoadClientTLSConfig builds an mTLS client config presenting a client certificate to the server
// and verifying the server certificate against the provided CA pool.
func LoadClientTLSConfig(caCertPath, clientCertPath, clientKeyPath, serverName string) (*tls.Config, error) {
	caCertBytes, err := os.ReadFile(caCertPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate from %s: %w", caCertPath, err)
	}

	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caCertBytes) {
		return nil, fmt.Errorf("failed to parse CA certificate into cert pool")
	}

	clientCert, err := tls.LoadX509KeyPair(clientCertPath, clientKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load client keypair (%s, %s): %w", clientCertPath, clientKeyPath, err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{clientCert},
		RootCAs:      caPool,
		ServerName:   serverName,
		MinVersion:   tls.VersionTLS13,
	}, nil
}

// NewServerTransportCredentials wraps LoadServerTLSConfig for gRPC servers.
func NewServerTransportCredentials(caCertPath, serverCertPath, serverKeyPath string) (credentials.TransportCredentials, error) {
	cfg, err := LoadServerTLSConfig(caCertPath, serverCertPath, serverKeyPath)
	if err != nil {
		return nil, err
	}
	return credentials.NewTLS(cfg), nil
}

// NewClientTransportCredentials wraps LoadClientTLSConfig for gRPC clients.
func NewClientTransportCredentials(caCertPath, clientCertPath, clientKeyPath, serverName string) (credentials.TransportCredentials, error) {
	cfg, err := LoadClientTLSConfig(caCertPath, clientCertPath, clientKeyPath, serverName)
	if err != nil {
		return nil, err
	}
	return credentials.NewTLS(cfg), nil
}

