package utils

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"time"
)

func GetFirstCertExpiryFromPEM(certPEM []byte) (time.Time, error) {
	block, _ := pem.Decode(certPEM)
	if block == nil || block.Type != "CERTIFICATE" {
		return time.Time{}, fmt.Errorf("failed to parse certificate PEM")
	}

	parsedCert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse certificate: %w", err)
	}

	return parsedCert.NotAfter, nil
}
