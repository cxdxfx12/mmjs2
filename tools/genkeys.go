//go:build ignore

package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"os"
)

func main() {
	// Generate RSA 2048 key pair
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}

	// Private key PEM
	privPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(priv),
	})

	// Public key PEM
	pubBytes, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		panic(err)
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	})

	// AES key (keep the same bytes but output as hex for PHP)
	aesKey := []byte{
		0x7a, 0x3b, 0x95, 0x1c, 0x4e, 0x62, 0xaf, 0x8d,
		0x2f, 0x51, 0xdc, 0x73, 0x1a, 0x88, 0x6b, 0x0e,
		0x39, 0x47, 0xb5, 0xe2, 0x14, 0x9d, 0x26, 0xf8,
		0x5c, 0x03, 0xea, 0x67, 0x90, 0x1f, 0xd4, 0xab,
	}

	fmt.Println("=== AES KEY (hex for PHP) ===")
	fmt.Println(hex.EncodeToString(aesKey))

	fmt.Println("\n=== RSA PRIVATE KEY (for server, KEEP SECRET) ===")
	fmt.Println(string(privPEM))

	fmt.Println("\n=== RSA PUBLIC KEY (for Go client) ===")
	fmt.Println(string(pubPEM))

	// Save to files
	os.WriteFile("server_private.key", privPEM, 0600)
	os.WriteFile("client_public.key", pubPEM, 0644)
	fmt.Println("\nKeys saved to server_private.key and client_public.key")
}
