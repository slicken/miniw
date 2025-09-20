package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/tyler-smith/go-bip32"
	"github.com/tyler-smith/go-bip39"
)

type bitcoin struct {
	name           string
	xpub           byte
	xpriv          byte
	isSegWit       bool
	isNative       bool
	isTaproot      bool
	derivationPath string
}

func (btc bitcoin) Name() string {
	return btc.name
}

var btcMap = map[string]bitcoin{
	"legacy":  {name: "bitcoin legacy", xpub: 0x00, xpriv: 0x80, isSegWit: false, isNative: false, isTaproot: false, derivationPath: "m/44'/0'/0'/0/0"},
	"segwit":  {name: "bitcoin segwit", xpub: 0x05, xpriv: 0x80, isSegWit: true, isNative: false, isTaproot: false, derivationPath: "m/49'/0'/0'/0/0"},
	"native":  {name: "bitcoin native", xpub: 0x04, xpriv: 0x80, isSegWit: true, isNative: true, isTaproot: false, derivationPath: "m/84'/0'/0'/0/0"},
	"taproot": {name: "bitcoin taproot", xpub: 0x04, xpriv: 0x80, isSegWit: true, isNative: true, isTaproot: true, derivationPath: "m/86'/0'/0'/0/0"},
}

func (btc bitcoin) getParams() *chaincfg.Params {
	param := &chaincfg.MainNetParams // Always use mainnet
	if btc.isSegWit {
		if btc.isNative {
			param.Bech32HRPSegwit = "bc" // Native SegWit Bech32 address prefix (mainnet)
		} else {
			param.PubKeyHashAddrID = 0x05 // SegWit P2SH address prefix (mainnet)
		}
		param.PrivateKeyID = 0x80 // SegWit private key prefix (mainnet)
	} else {
		param.PubKeyHashAddrID = 0x00 // Legacy P2PKH address prefix (mainnet)
		param.PrivateKeyID = 0x80     // Legacy private key prefix (mainnet)
	}
	return param
}

func (btc bitcoin) createPrivateKey() (*btcutil.WIF, error) {
	// Generate a new private key (mainnet)
	secret, err := btcec.NewPrivateKey()
	if err != nil {
		return nil, err
	}

	// Ensure it's for mainnet (not testnet) with the correct prefix
	return btcutil.NewWIF(secret, btc.getParams(), true)
}

func (btc bitcoin) getAddress(wif *btcutil.WIF) (btcutil.Address, error) {
	pubKey := wif.PrivKey.PubKey()

	// Generate Taproot address (starts with 'bc1p')
	if btc.isTaproot {
		// Compute the Taproot output key
		taprootKey := txscript.ComputeTaprootKeyNoScript(pubKey)
		// Extract the x-coordinate of the public key (32 bytes)
		taprootKeyBytes := taprootKey.X().Bytes()
		// Create the Taproot address
		addr, err := btcutil.NewAddressTaproot(taprootKeyBytes, btc.getParams())
		if err != nil {
			return nil, err
		}
		return addr, nil
	}

	// Generate Native Segwit (starts with 'bc1')
	if btc.isNative {
		addr, err := btcutil.NewAddressWitnessPubKeyHash(btcutil.Hash160(pubKey.SerializeCompressed()), btc.getParams())
		if err != nil {
			return nil, err
		}
		return addr, nil
	}

	// Generate Segwit integrated withness (starts with '3')
	if btc.isSegWit {
		addr, err := btcutil.NewAddressScriptHash(btcutil.Hash160(pubKey.SerializeCompressed()), btc.getParams())
		if err != nil {
			return nil, err
		}
		return addr, nil
	}

	// Otherwise, generate a legacy P2PKH address (starts with '1')
	addr, err := btcutil.NewAddressPubKeyHash(btcutil.Hash160(pubKey.SerializeCompressed()), btc.getParams())
	if err != nil {
		return nil, err
	}
	return addr, nil
}

func (btc bitcoin) GenerateKeys() (*KeyPair, error) {
	if btc.name == "" {
		return nil, errors.New("network not found")
	}

	var privateKey *btcutil.WIF
	var mnemonic string
	var err error

	// Check for custom private key first
	if customPrivate != "" {
		// Decode WIF private key
		privateKey, err = btcutil.DecodeWIF(customPrivate)
		if err != nil {
			return nil, fmt.Errorf("failed to decode WIF private key: %v", err)
		}

		// Validate network parameters
		if !privateKey.IsForNet(btc.getParams()) {
			return nil, fmt.Errorf("private key is not for mainnet")
		}

		// If -a/--all flag is set, we should try to derive the mnemonic
		if *infoFlag || *infoLongFlag {
			// Note: We can't derive mnemonic from private key
			mnemonic = "(cannot derive mnemonic from private key)"
			btc.derivationPath = "(cannot derive path from private key)"
		}
	} else if customMnemonic != "" {
		mnemonic = customMnemonic
		// Generate seed from the custom mnemonic
		seed := bip39.NewSeed(mnemonic, "")
		masterKey, err := bip32.NewMasterKey(seed)
		if err != nil {
			return nil, err
		}

		// If a custom derivation path is provided, use it
		derivationPath := btc.derivationPath
		if customPath != "" {
			derivationPath = customPath
		}

		// Derive the child key using the specified path
		childKey, err := btc.deriveChildKeyFromMaster(masterKey, derivationPath)
		if err != nil {
			return nil, err
		}

		// Convert the child key to WIF
		privKey, _ := btcec.PrivKeyFromBytes(childKey.Key)
		privateKey, err = btcutil.NewWIF(privKey, btc.getParams(), true)
		if err != nil {
			return nil, err
		}

		// Update the derivation path in the KeyPair
		btc.derivationPath = derivationPath
	} else if *infoFlag || *infoLongFlag {
		// If mnemonic flag is used, generate a random mnemonic
		entropy, err := bip39.NewEntropy(256)
		if err != nil {
			return nil, err
		}
		mnemonic, err = bip39.NewMnemonic(entropy)
		if err != nil {
			return nil, err
		}
		seed := bip39.NewSeed(mnemonic, "")
		masterKey, err := bip32.NewMasterKey(seed)
		if err != nil {
			return nil, err
		}

		// Derive the child key using the default derivation path
		childKey, err := btc.deriveChildKeyFromMaster(masterKey, btc.derivationPath)
		if err != nil {
			return nil, err
		}

		// Convert the child key to WIF
		privKey, _ := btcec.PrivKeyFromBytes(childKey.Key)
		privateKey, err = btcutil.NewWIF(privKey, btc.getParams(), true)
		if err != nil {
			return nil, err
		}
	} else {
		// Generate a new private key if no custom options are provided
		privateKey, err = btc.createPrivateKey()
		if err != nil {
			return nil, err
		}
	}

	// Derive the address from the private key
	address, err := btc.getAddress(privateKey)
	if err != nil {
		return nil, err
	}

	k := new(KeyPair)
	k.network = btc.name
	k.private = privateKey.String()
	k.public = address.EncodeAddress()
	// Only include mnemonic and path if -a/--all is set
	if *infoFlag || *infoLongFlag {
		k.mnemonic = mnemonic
		k.derivationPath = btc.derivationPath
	}

	return k, nil
}

func (btc bitcoin) deriveChildKeyFromMaster(masterKey *bip32.Key, path string) (*bip32.Key, error) {
	// Split the derivation path into components
	components := strings.Split(path, "/")
	// Remove the first "m" part of the path
	components = components[1:]

	// Iterate through the path components and derive child keys
	currentKey := masterKey
	for _, component := range components {
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
			return nil, fmt.Errorf("error deriving child key: %v", err)
		}
	}

	return currentKey, nil
}
