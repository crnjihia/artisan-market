// path: scripts/generate-certs.go
package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

func main() {
	outDir := "certs"
	if len(os.Args) > 1 {
		outDir = os.Args[1]
	}

	if err := os.MkdirAll(outDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create certs dir: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("==> Generating Root CA (ArtisanMarketRootCA)...")
	caPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		panic(err)
	}

	caTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2026),
		Subject: pkix.Name{
			Country:            []string{"KE"},
			Province:           []string{"Nairobi"},
			Locality:           []string{"Nairobi"},
			Organization:       []string{"Artisan Market"},
			OrganizationalUnit: []string{"Security CA"},
			CommonName:         "ArtisanMarketRootCA",
		},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().AddDate(2, 0, 0),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}

	caBytes, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caPrivKey.PublicKey, caPrivKey)
	if err != nil {
		panic(err)
	}

	savePEM(filepath.Join(outDir, "ca.crt"), "CERTIFICATE", caBytes)
	savePEM(filepath.Join(outDir, "ca.key"), "RSA PRIVATE KEY", x509.MarshalPKCS1PrivateKey(caPrivKey))

	services := []string{"gateway", "user", "product", "order", "payment"}
	for _, svc := range services {
		fmt.Printf("==> Generating mTLS certificate for %s...\n", svc)
		svcKey, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			panic(err)
		}

		serialNumber, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
		svcTemplate := &x509.Certificate{
			SerialNumber: serialNumber,
			Subject: pkix.Name{
				Country:            []string{"KE"},
				Province:           []string{"Nairobi"},
				Locality:           []string{"Nairobi"},
				Organization:       []string{"Artisan Market"},
				OrganizationalUnit: []string{svc},
				CommonName:         svc,
			},
			NotBefore:    time.Now().Add(-1 * time.Hour),
			NotAfter:     time.Now().AddDate(1, 0, 0),
			SubjectKeyId: []byte{1, 2, 3, 4, 6},
			ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
			KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
			DNSNames: []string{
				svc,
				svc + ".artisan-market.svc.cluster.local",
				"localhost",
			},
			IPAddresses: []net.IP{
				net.ParseIP("127.0.0.1"),
				net.ParseIP("::1"),
			},
		}

		svcBytes, err := x509.CreateCertificate(rand.Reader, svcTemplate, caTemplate, &svcKey.PublicKey, caPrivKey)
		if err != nil {
			panic(err)
		}

		savePEM(filepath.Join(outDir, svc+".crt"), "CERTIFICATE", svcBytes)
		savePEM(filepath.Join(outDir, svc+".key"), "RSA PRIVATE KEY", x509.MarshalPKCS1PrivateKey(svcKey))
	}

	fmt.Printf("==> Successfully generated all mTLS certificates in %s/\n", outDir)
}

func savePEM(path, blockType string, bytes []byte) {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	if err := pem.Encode(file, &pem.Block{Type: blockType, Bytes: bytes}); err != nil {
		panic(err)
	}
}

