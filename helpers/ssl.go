package helpers

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"time"

	"gitlab.dev.ict/golang/libs/utils"
)

const (
	// PathToCert is the path to the certificate file
	DefFileCert = "cert.pem"
	// PathToKey is the path to the private key file
	DefFileKey = "key.pem"
)

func GenerateSSLCert(pathToCert, pathToKey string) (cert tls.Certificate, err error) {
	if utils.IsExists(pathToCert) && utils.IsExists(pathToKey) {
		cert, err = tls.LoadX509KeyPair(pathToCert, pathToKey)
	} else {
		cert, err = GenerateSelfSignedCertificate(pathToCert, pathToKey)
	}
	return
}

func GenerateSelfSignedCertificate(pathToCert, pathToKey string) (tls.Certificate, error) {
	// Generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, err
	}

	// Generate self-signed certificate
	subject := pkix.Name{
		Organization: []string{"Lifecell"},
		CommonName:   "ai.dev.ict",
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      subject,
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour), // 1 year
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP("10.1.60.162")},
		DNSNames:     []string{"ai.dev.ict"},
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return tls.Certificate{}, err
	}

	// Encode certificate and private key to PEM format
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})

	// Save cert and key to files
	_ = os.WriteFile(pathToCert, certPEM, 0644)
	_ = os.WriteFile(pathToKey, keyPEM, 0644)

	// Create tls.Certificate from PEM data
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return tls.Certificate{}, err
	}

	return cert, nil
}
