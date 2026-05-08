// scripts/genkey — generates an Ed25519 key pair for the vote simulator.
//
// Prints:
//   SIMULATOR_ED25519_PRIVATE_KEY=<base64>   → paste into .env
//   SIMULATOR_ED25519_PUBLIC_KEY=<base64>    → seeded into province_keys by make seed
//
// The public key must match the value seeded into province_keys for all provinces.
// The seed migration reads SIMULATOR_ED25519_PUBLIC_KEY from the environment.

package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"log"
)

func main() {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		log.Fatalf("generate Ed25519 key: %v", err)
	}

	privB64 := base64.StdEncoding.EncodeToString(priv)
	pubB64 := base64.StdEncoding.EncodeToString(pub)

	fmt.Println("# Add these to your .env file:")
	fmt.Printf("SIMULATOR_ED25519_PRIVATE_KEY=%s\n", privB64)
	fmt.Printf("SIMULATOR_ED25519_PUBLIC_KEY=%s\n", pubB64)
	fmt.Println()
	fmt.Println("# Re-run 'make seed' after setting SIMULATOR_ED25519_PUBLIC_KEY")
	fmt.Println("# so province_keys are updated with the new key.")
}
