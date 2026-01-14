// Package tls provides TLS certificate management for FlyDB.
// It handles certificate generation, validation, storage, and rotation.
package tls

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

// CertConfig holds configuration for certificate generation.
type CertConfig struct {
	// Organization name for the certificate
	Organization string
	// Common name (hostname) for the certificate
	CommonName string
	// Validity period in days
	ValidityDays int
	// Key size (256 or 384 for ECDSA)
	KeySize int
	// Subject Alternative Names (SANs)
	SANs []string
}

// DefaultCertConfig returns default certificate configuration.
func DefaultCertConfig() CertConfig {
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "localhost"
	}
	
	return CertConfig{
		Organization: "FlyDB",
		CommonName:   hostname,
		ValidityDays: 365,
		KeySize:      256,
		SANs:         []string{hostname, "localhost", "127.0.0.1", "::1"},
	}
}

// GenerateSelfSignedCert generates a self-signed certificate and private key.
func GenerateSelfSignedCert(config CertConfig) (certPEM, keyPEM []byte, err error) {
	// Generate private key
	var priv *ecdsa.PrivateKey
	switch config.KeySize {
	case 256:
		priv, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	case 384:
		priv, err = ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	default:
		return nil, nil, fmt.Errorf("unsupported key size: %d (use 256 or 384)", config.KeySize)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	// Generate serial number
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	// Create certificate template
	notBefore := time.Now()
	notAfter := notBefore.Add(time.Duration(config.ValidityDays) * 24 * time.Hour)

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{config.Organization},
			CommonName:   config.CommonName,
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	// Add SANs
	for _, san := range config.SANs {
		template.DNSNames = append(template.DNSNames, san)
	}

	// Create self-signed certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	// Encode certificate to PEM
	certPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	// Encode private key to PEM
	privBytes, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal private key: %w", err)
	}
	keyPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: privBytes,
	})

	return certPEM, keyPEM, nil
}

// SaveCertificates saves certificate and key to files with appropriate permissions.
func SaveCertificates(certPath, keyPath string, certPEM, keyPEM []byte) error {
	// Create directory if it doesn't exist
	certDir := filepath.Dir(certPath)
	if err := os.MkdirAll(certDir, 0755); err != nil {
		return fmt.Errorf("failed to create certificate directory: %w", err)
	}

	// Write certificate file (readable by all)
	if err := os.WriteFile(certPath, certPEM, 0644); err != nil {
		return fmt.Errorf("failed to write certificate file: %w", err)
	}

	// Write key file (readable only by owner)
	if err := os.WriteFile(keyPath, keyPEM, 0600); err != nil {
		return fmt.Errorf("failed to write key file: %w", err)
	}

	return nil
}

// ValidateCertificate validates a certificate file and checks expiration.
func ValidateCertificate(certPath string) error {
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return fmt.Errorf("failed to read certificate: %w", err)
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		return fmt.Errorf("failed to decode PEM block")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Check if certificate is expired
	now := time.Now()
	if now.Before(cert.NotBefore) {
		return fmt.Errorf("certificate is not yet valid (valid from %s)", cert.NotBefore)
	}
	if now.After(cert.NotAfter) {
		return fmt.Errorf("certificate expired on %s", cert.NotAfter)
	}

	// Warn if certificate expires soon (within 30 days)
	daysUntilExpiry := int(cert.NotAfter.Sub(now).Hours() / 24)
	if daysUntilExpiry <= 30 {
		fmt.Fprintf(os.Stderr, "Warning: Certificate expires in %d days (%s)\n", daysUntilExpiry, cert.NotAfter)
	}

	return nil
}

// LoadTLSConfig loads TLS configuration from certificate and key files.
func LoadTLSConfig(certPath, keyPath string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load TLS certificate: %w", err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
		},
		PreferServerCipherSuites: true,
	}, nil
}

// GetDefaultCertPaths returns default certificate paths based on user privileges.
func GetDefaultCertPaths() (certDir, certPath, keyPath string) {
	if os.Getuid() == 0 {
		// Running as root - use system directory
		certDir = "/etc/flydb/certs"
	} else {
		// Running as user - use user config directory
		home, err := os.UserHomeDir()
		if err != nil {
			certDir = "./certs"
		} else {
			certDir = filepath.Join(home, ".config", "flydb", "certs")
		}
	}

	certPath = filepath.Join(certDir, "server.crt")
	keyPath = filepath.Join(certDir, "server.key")
	return
}

// EnsureCertificates ensures that valid certificates exist, generating them if necessary.
func EnsureCertificates(certPath, keyPath string, config CertConfig) error {
	// Check if certificates already exist
	if _, err := os.Stat(certPath); err == nil {
		if _, err := os.Stat(keyPath); err == nil {
			// Certificates exist, validate them
			if err := ValidateCertificate(certPath); err == nil {
				// Certificates are valid
				return nil
			}
			// Certificates are invalid or expired, regenerate
			fmt.Fprintf(os.Stderr, "Warning: Existing certificates are invalid, regenerating...\n")
		}
	}

	// Generate new certificates
	fmt.Fprintf(os.Stderr, "Generating self-signed TLS certificates...\n")
	certPEM, keyPEM, err := GenerateSelfSignedCert(config)
	if err != nil {
		return fmt.Errorf("failed to generate certificates: %w", err)
	}

	// Save certificates
	if err := SaveCertificates(certPath, keyPath, certPEM, keyPEM); err != nil {
		return fmt.Errorf("failed to save certificates: %w", err)
	}

	fmt.Fprintf(os.Stderr, "TLS certificates generated successfully:\n")
	fmt.Fprintf(os.Stderr, "  Certificate: %s\n", certPath)
	fmt.Fprintf(os.Stderr, "  Private Key: %s\n", keyPath)
	fmt.Fprintf(os.Stderr, "\nNote: These are self-signed certificates for development/testing.\n")
	fmt.Fprintf(os.Stderr, "For production, use certificates from a trusted CA.\n")

	return nil
}
