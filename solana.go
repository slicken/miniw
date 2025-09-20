package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/blocto/solana-go-sdk/types"
	"github.com/btcsuite/btcd/btcutil/base58"
	"github.com/tyler-smith/go-bip32"
	"github.com/tyler-smith/go-bip39"
)

type solana struct{}

func (sol solana) Name() string {
	return "Solana"
}

// GenerateKeys for Solana
func (sol solana) GenerateKeys() (*KeyPair, error) {
	var wallet types.Account
	var mnemonic string
	var derivationPath string
	var err error

	// Check for custom private key first
	if customPrivate != "" {
		// Decode base58 private key
		privateKeyBytes := base58.Decode(customPrivate)
		if len(privateKeyBytes) != 64 {
			return nil, fmt.Errorf("invalid private key length: expected 64 bytes")
		}

		// Create Solana account from private key
		wallet, err = types.AccountFromBytes(privateKeyBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to create account from private key: %v", err)
		}

		// If -a/--all flag is set, add placeholder messages
		if *infoFlag || *infoLongFlag {
			mnemonic = "(cannot derive mnemonic from private key)"
			derivationPath = "(cannot derive path from private key)"
		}
	} else if customMnemonic != "" {
		// Validate the mnemonic
		if !bip39.IsMnemonicValid(customMnemonic) {
			return nil, fmt.Errorf("invalid mnemonic phrase")
		}

		// Use the custom mnemonic
		mnemonic = customMnemonic

		// Generate seed from mnemonic
		seed := bip39.NewSeed(mnemonic, "") // No passphrase for simplicity

		// Derive the private key using BIP-44 derivation path
		privateKey, err := deriveSolanaPrivateKey(seed, customPath)
		if err != nil {
			return nil, fmt.Errorf("failed to derive private key: %v", err)
		}

		// Create a Solana wallet from the private key
		wallet, err = types.AccountFromSeed(privateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to create wallet from seed: %v", err)
		}

		// Set the derivation path
		derivationPath = customPath
		if derivationPath == "" {
			derivationPath = "m/44'/501'/0'/0'" // Default Solana BIP-44 derivation path
		}
	} else if *infoFlag || *infoLongFlag {
		// Generate a new mnemonic
		entropy, err := bip39.NewEntropy(128) // 128 bits of entropy for a 12-word mnemonic
		if err != nil {
			return nil, fmt.Errorf("failed to generate entropy: %v", err)
		}

		mnemonic, err = bip39.NewMnemonic(entropy)
		if err != nil {
			return nil, fmt.Errorf("failed to generate mnemonic: %v", err)
		}

		// Generate seed from mnemonic
		seed := bip39.NewSeed(mnemonic, "")

		// Derive the private key using default BIP-44 derivation path
		privateKey, err := deriveSolanaPrivateKey(seed, "")
		if err != nil {
			return nil, fmt.Errorf("failed to derive private key: %v", err)
		}

		// Create a Solana wallet from the private key
		wallet, err = types.AccountFromSeed(privateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to create wallet from seed: %v", err)
		}

		derivationPath = "m/44'/501'/0'/0'"
	} else {
		// Generate a new wallet with a random mnemonic (not exposed)
		wallet = types.NewAccount()
	}

	// Create the KeyPair
	k := new(KeyPair)
	k.network = "solana"
	k.private = base58.Encode(wallet.PrivateKey)
	k.public = wallet.PublicKey.ToBase58()
	// Only include mnemonic and path if -a/--all is set
	if *infoFlag || *infoLongFlag {
		k.mnemonic = mnemonic
		k.derivationPath = derivationPath
	}

	return k, nil
}

// deriveSolanaPrivateKey derives the private key from the seed using the BIP-44 derivation path
func deriveSolanaPrivateKey(seed []byte, customPath string) ([]byte, error) {
	// Default Solana BIP-44 derivation path
	derivationPath := "m/44'/501'/0'/0'"
	if customPath != "" {
		derivationPath = customPath
	}

	// Split the derivation path into components
	components := strings.Split(derivationPath, "/")
	if components[0] != "m" {
		return nil, fmt.Errorf("invalid derivation path: must start with 'm'")
	}

	// Create the master key from the seed
	masterKey, err := bip32.NewMasterKey(seed)
	if err != nil {
		return nil, fmt.Errorf("failed to create master key: %v", err)
	}

	// Iterate through the path components and derive child keys
	currentKey := masterKey
	for _, component := range components[1:] {
		// Check if the component is hardened (ends with a single quote)
		hardened := false
		if strings.HasSuffix(component, "'") {
			hardened = true
			component = strings.TrimSuffix(component, "'")
		}

		// Convert component to uint32
		index, err := strconv.ParseUint(component, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid index in derivation path: %v", err)
		}

		// Harden the index if needed (add 0x80000000 for hardened)
		if hardened {
			index += 0x80000000
		}

		// Derive the child key at this index
		currentKey, err = currentKey.NewChildKey(uint32(index))
		if err != nil {
			return nil, fmt.Errorf("failed to derive child key: %v", err)
		}
	}

	// Return the private key bytes
	return currentKey.Key, nil
}

func (sol solana) GenerateFromPrivateKey(privateKeyBase58 string) (*KeyPair, error) {
	// Decode base58 private key
	privateKeyBytes := base58.Decode(privateKeyBase58)
	if len(privateKeyBytes) != 64 {
		return nil, fmt.Errorf("invalid private key length: expected 64 bytes")
	}

	// Create Solana account from private key
	wallet, err := types.AccountFromBytes(privateKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to create account from private key: %v", err)
	}

	return &KeyPair{
		network: "solana",
		public:  wallet.PublicKey.ToBase58(),
		private: privateKeyBase58,
	}, nil
}
