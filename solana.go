package main

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"

	"github.com/blocto/solana-go-sdk/types"
	"github.com/btcsuite/btcd/btcutil/base58"
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
			derivationPath = solanaDefaultDerivationPath
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

		derivationPath = solanaDefaultDerivationPath
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

const (
	solanaDefaultDerivationPath = "m/44'/501'/0'/0'"
	hardenedKeyStart            = uint32(0x80000000)
)

// deriveSolanaPrivateKey derives the Ed25519 seed using SLIP-0010 hardened derivation.
func deriveSolanaPrivateKey(seed []byte, customPath string) ([]byte, error) {
	derivationPath := solanaDefaultDerivationPath
	if customPath != "" {
		derivationPath = customPath
	}

	// Split the derivation path into components
	components := strings.Split(derivationPath, "/")
	if components[0] != "m" {
		return nil, fmt.Errorf("invalid derivation path: must start with 'm'")
	}

	key, chainCode := deriveSolanaMasterKey(seed)

	// Iterate through the path components and derive child keys
	for _, component := range components[1:] {
		// Check if the component is hardened (ends with a single quote)
		if !strings.HasSuffix(component, "'") {
			return nil, fmt.Errorf("invalid derivation path: Solana Ed25519 derivation requires hardened segments")
		}
		component = strings.TrimSuffix(component, "'")

		// Convert component to uint32
		index, err := strconv.ParseUint(component, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid index in derivation path: %v", err)
		}
		if index >= uint64(hardenedKeyStart) {
			return nil, fmt.Errorf("invalid index in derivation path: %d is too large", index)
		}

		key, chainCode = deriveSolanaChildKey(key, chainCode, uint32(index)+hardenedKeyStart)
	}

	return key, nil
}

func deriveSolanaMasterKey(seed []byte) ([]byte, []byte) {
	return splitSolanaHMAC([]byte("ed25519 seed"), seed)
}

func deriveSolanaChildKey(key []byte, chainCode []byte, index uint32) ([]byte, []byte) {
	data := make([]byte, 1+len(key)+4)
	copy(data[1:], key)
	binary.BigEndian.PutUint32(data[len(data)-4:], index)

	return splitSolanaHMAC(chainCode, data)
}

func splitSolanaHMAC(hmacKey []byte, data []byte) ([]byte, []byte) {
	mac := hmac.New(sha512.New, hmacKey)
	mac.Write(data)
	sum := mac.Sum(nil)

	return sum[:32], sum[32:]
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
