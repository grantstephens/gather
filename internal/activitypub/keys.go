package activitypub

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
)

func GenerateKeyPair() (privateKey string, publicKey string, err error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", err
	}

	privateBytes := x509.MarshalPKCS1PrivateKey(key)
	privatePem := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateBytes,
	})

	publicBytes, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return "", "", err
	}
	publicPem := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicBytes,
	})

	return string(privatePem), string(publicPem), nil
}

func ParsePrivateKey(pemData string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}
