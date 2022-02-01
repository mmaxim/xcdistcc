package main

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"os"

	"golang.org/x/crypto/nacl/box"
)

func main() {
	priv, public, err := box.GenerateKey(rand.Reader)
	if err != nil {
		log.Printf("failed to generate key: %s", err)
		os.Exit(3)
	}
	log.Printf("public: %s", hex.EncodeToString(public[:]))
	log.Printf("private: %s", hex.EncodeToString(priv[:]))
	os.Exit(0)
}
