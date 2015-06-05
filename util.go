package neptulon

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	mathrand "math/rand"
	"net"
	"strings"
	"time"
)

var letters = []rune(". !abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

// randString generates a random string sequence of given size.
func randString(n int) string {
	mathrand.Seed(time.Now().UTC().UnixNano())
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[mathrand.Intn(len(letters))]
	}
	return string(b)
}

// getID generates a unique ID using crypto/rand in the form "m-96bitBase16" and total of 26 characters long (i.e. m-18dc2ae3898820d9c5df4f38).
func getID() (string, error) {
	// todo: we can use sequential numbers optionally, just as the Android client does (1, 2, 3..) in upstream messages
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("m-%x", b), nil
}

// genCert generates a PEM encoded X.509 certificate and private key pair (i.e. 'cert.pem', 'key.pem').
// This code is based on the sample from http://golang.org/src/crypto/tls/generate_cert.go (taken at Jan 30, 2015).
// If no private key is provided, the certificate is marked as self-signed CA.
// host = Comma-separated hostnames and IPs to generate a certificate for. i.e. "localhost,127.0.0.1"
// validFor = Validity period for the certificate. Defaults to time.Duration max (290 years).
// ca, caPriv = CA certificate/private key to sign the new certificate. If not given, the generated certificate will be a self-signed CA.
// keyLength = Key length for the new certificate. Defaults to 3248 bits RSA key.
// cn, org = Common name and organization fields of the certificate.
func genCert(host string, validFor time.Duration, ca *x509.Certificate, caPriv *rsa.PrivateKey, keyLength int, cn, org string) (pemBytes, privBytes []byte, err error) {
	isCA := ca == nil
	hosts := strings.Split(host, ",")
	if keyLength == 0 {
		keyLength = 3248
	}
	privKey, err := rsa.GenerateKey(rand.Reader, keyLength)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate certificate private RSA key: %v", err)
	}
	notBefore := time.Now()
	notAfter := notBefore.Add(290 * 365 * 24 * time.Hour) //290 years
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate the certificate serial number: %v", err)
	}
	if validFor != 0 {
		notAfter = notBefore.Add(validFor)
	}

	cert := x509.Certificate{
		IsCA:         isCA,
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   cn,
			Organization: []string{org},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}

	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			cert.IPAddresses = append(cert.IPAddresses, ip)
		} else {
			cert.DNSNames = append(cert.DNSNames, h)
		}
	}

	signerCert := &cert
	signerPriv := privKey
	if isCA {
		cert.KeyUsage |= x509.KeyUsageCertSign
		cert.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth}
	} else {
		cert.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
		signerCert = ca
		signerPriv = caPriv
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &cert, signerCert, &privKey.PublicKey, signerPriv)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create certificate: %v", err)
	}

	pemBytes = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	privBytes = pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privKey)})
	return
}
