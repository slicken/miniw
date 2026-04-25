package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
)

type BitcoinAPI struct {
	BaseURL         string
	UTXOPath        string
	BroadcastPath   string
	BroadcastFormat string // "raw" or "json"
}

var bitcoin_apis = []BitcoinAPI{
	{
		BaseURL:         "https://mempool.space/api",
		UTXOPath:        "/address/%s/utxo",
		BroadcastPath:   "/tx",
		BroadcastFormat: "raw",
	},
	{
		BaseURL:         "https://blockstream.info/api",
		UTXOPath:        "/address/%s/utxo",
		BroadcastPath:   "/tx",
		BroadcastFormat: "json",
	},
}

// UTXO represents an unspent transaction output
type UTXO struct {
	TxID   string `json:"txid"`
	Vout   uint32 `json:"vout"`
	Value  int64  `json:"value"` // Value in satoshis
	Script string `json:"scriptPubKey"`
}

// BitcoinRPC client
type BitcoinRPC struct {
	client *http.Client
	api    BitcoinAPI
}

// BitcoinBalance represents the balance information for a Bitcoin address
type BitcoinBalance struct {
	Address string  `json:"address"`
	Balance float64 `json:"balance"` // Balance in BTC
	UTXOs   []UTXO  `json:"utxos"`
}

// GetBitcoinBalance retrieves the balance and UTXOs for a Bitcoin address
func GetBitcoinBalance(address string, addressType string, customRPC string) (*BitcoinBalance, error) {
	var apisToTry []BitcoinAPI

	if customRPC != "" {
		// Use custom RPC endpoint
		apisToTry = []BitcoinAPI{
			{
				BaseURL:         customRPC,
				UTXOPath:        "/address/%s/utxo",
				BroadcastPath:   "/tx",
				BroadcastFormat: "raw",
			},
		}
	} else {
		// Use default Bitcoin APIs
		apisToTry = bitcoin_apis
	}

	// Try the APIs
	for _, api := range apisToTry {
		rpc := &BitcoinRPC{
			client: &http.Client{Timeout: 10 * time.Second},
			api:    api,
		}

		balance, utxos, err := rpc.getBitcoinBalance(address)
		if err == nil {
			return &BitcoinBalance{
				Address: address,
				Balance: balance,
				UTXOs:   utxos,
			}, nil
		}
	}
	return nil, fmt.Errorf("failed to get balance from all Bitcoin APIs")
}

// SendBitcoin sends Bitcoin from one address to another
func SendBitcoin(privateKeyWIF, fromAddress, toAddress string, amount string, addressType string, customRPC string) (string, error) {
	// Parse the private key from WIF format
	wif, err := btcutil.DecodeWIF(privateKeyWIF)
	if err != nil {
		return "", fmt.Errorf("failed to decode WIF private key: %v", err)
	}

	// Parse destination address
	destAddress, err := btcutil.DecodeAddress(toAddress, &chaincfg.MainNetParams)
	if err != nil {
		return "", fmt.Errorf("invalid destination address: %v", err)
	}

	// Parse source address
	sourceAddress, err := btcutil.DecodeAddress(fromAddress, &chaincfg.MainNetParams)
	if err != nil {
		return "", fmt.Errorf("invalid source address: %v", err)
	}

	// Determine which APIs to try
	var apisToTry []BitcoinAPI
	if customRPC != "" {
		// Use custom RPC endpoint
		apisToTry = []BitcoinAPI{
			{
				BaseURL:         customRPC,
				UTXOPath:        "/address/%s/utxo",
				BroadcastPath:   "/tx",
				BroadcastFormat: "raw",
			},
		}
	} else {
		// Use default Bitcoin APIs
		apisToTry = bitcoin_apis
	}

	// Try the APIs
	for _, api := range apisToTry {
		rpc := &BitcoinRPC{
			client: &http.Client{Timeout: 30 * time.Second},
			api:    api,
		}

		// Get balance and UTXOs
		balance, utxos, err := rpc.getBitcoinBalance(fromAddress)
		if err != nil {
			continue
		}

		if balance <= 0 {
			return "", fmt.Errorf("no balance found for address")
		}

		// Send transaction
		txHash, err := rpc.sendBitcoinTransaction(wif, sourceAddress, destAddress, amount, utxos, addressType)
		if err == nil {
			return txHash, nil
		}
	}

	return "", fmt.Errorf("failed to send transaction from all Bitcoin APIs")
}

func (rpc *BitcoinRPC) getBitcoinBalance(address string) (float64, []UTXO, error) {
	// REAL Bitcoin blockchain connection
	url := fmt.Sprintf("%s%s", rpc.api.BaseURL, fmt.Sprintf(rpc.api.UTXOPath, address))

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to create request: %v", err)
	}

	resp, err := rpc.client.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to connect to Bitcoin network: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, nil, fmt.Errorf("bitcoin API error: %d", resp.StatusCode)
	}

	// Parse UTXOs from blockchain
	var utxos []UTXO
	if err := json.NewDecoder(resp.Body).Decode(&utxos); err != nil {
		return 0, nil, fmt.Errorf("failed to parse Bitcoin response: %v", err)
	}

	// Calculate total balance from UTXOs (convert satoshis to BTC)
	var totalBalance float64
	for _, utxo := range utxos {
		totalBalance += float64(utxo.Value) / btcutil.SatoshiPerBitcoin
	}

	return totalBalance, utxos, nil
}

func (rpc *BitcoinRPC) sendBitcoinTransaction(wif *btcutil.WIF, fromAddress, destAddress btcutil.Address, amount string, utxos []UTXO, addressType string) (string, error) {
	// Create REAL Bitcoin transaction
	tx := wire.NewMsgTx(wire.TxVersion)

	// Convert amount to satoshis
	amountSatoshis, err := parseAmountToInt64(amount, 8)
	if err != nil {
		return "", fmt.Errorf("invalid amount: %v", err)
	}

	// Add inputs from real UTXOs
	var totalInputValue int64
	for _, utxo := range utxos {
		prevHash, err := chainhash.NewHashFromStr(utxo.TxID)
		if err != nil {
			return "", fmt.Errorf("invalid UTXO hash: %v", err)
		}

		prevOut := wire.NewOutPoint(prevHash, utxo.Vout)
		txIn := wire.NewTxIn(prevOut, nil, nil)
		tx.AddTxIn(txIn)
		totalInputValue += utxo.Value
	}

	// Add output to destination
	destScript, err := txscript.PayToAddrScript(destAddress)
	if err != nil {
		return "", fmt.Errorf("failed to create destination script: %v", err)
	}

	// Calculate fee (use a reasonable fee)
	fee := int64(10000) // 10,000 satoshis fee
	changeAmount := totalInputValue - amountSatoshis - fee

	if changeAmount < 0 {
		return "", fmt.Errorf("insufficient balance for transaction and fee")
	}

	// Add destination output
	txOut := wire.NewTxOut(amountSatoshis, destScript)
	tx.AddTxOut(txOut)

	// Add change output if needed
	if changeAmount > 0 {
		changeScript, err := txscript.PayToAddrScript(fromAddress)
		if err != nil {
			return "", fmt.Errorf("failed to create change script: %v", err)
		}
		changeTxOut := wire.NewTxOut(changeAmount, changeScript)
		tx.AddTxOut(changeTxOut)
	}

	// Sign REAL transaction
	err = rpc.signBitcoinTransaction(tx, wif, fromAddress, addressType, utxos)
	if err != nil {
		return "", fmt.Errorf("failed to sign transaction: %v", err)
	}

	// Broadcast REAL transaction to Bitcoin network
	txHash, err := rpc.broadcastTransaction(tx)
	if err != nil {
		return "", fmt.Errorf("failed to broadcast transaction: %v", err)
	}

	log.Printf("Bitcoin transaction sent - Amount: %s BTC, Fee: %f BTC, TXID: %s",
		amount, float64(fee)/btcutil.SatoshiPerBitcoin, txHash)

	return txHash, nil
}

func (rpc *BitcoinRPC) signBitcoinTransaction(tx *wire.MsgTx, wif *btcutil.WIF, address btcutil.Address, addressType string, utxos []UTXO) error {
	sourcePkScript, err := txscript.PayToAddrScript(address)
	if err != nil {
		return fmt.Errorf("failed to create source script: %v", err)
	}

	prevOutFetcher := txscript.NewMultiPrevOutFetcher(nil)
	for i, txIn := range tx.TxIn {
		prevOutFetcher.AddPrevOut(txIn.PreviousOutPoint, wire.NewTxOut(utxos[i].Value, sourcePkScript))
	}
	sigHashes := txscript.NewTxSigHashes(tx, prevOutFetcher)

	for i, txIn := range tx.TxIn {
		utxo := utxos[i]

		switch addressType {
		case "legacy":
			sigScript, err := txscript.SignatureScript(tx, i, sourcePkScript, txscript.SigHashAll, wif.PrivKey, true)
			if err != nil {
				return fmt.Errorf("failed to create legacy signature script for input %d: %v", i, err)
			}
			txIn.SignatureScript = sigScript
		case "segwit":
			redeemScript, err := p2wpkhRedeemScript(wif)
			if err != nil {
				return fmt.Errorf("failed to create redeem script for input %d: %v", i, err)
			}
			sigScript, err := txscript.NewScriptBuilder().AddData(redeemScript).Script()
			if err != nil {
				return fmt.Errorf("failed to create segwit signature script for input %d: %v", i, err)
			}
			txIn.SignatureScript = sigScript
			if err := createP2WPKHWitness(tx, sigHashes, wif, i, utxo); err != nil {
				return fmt.Errorf("failed to create segwit witness for input %d: %v", i, err)
			}
		case "native_segwit":
			txIn.SignatureScript = nil
			if err := createP2WPKHWitness(tx, sigHashes, wif, i, utxo); err != nil {
				return fmt.Errorf("failed to create native segwit witness for input %d: %v", i, err)
			}
		case "taproot":
			txIn.SignatureScript = nil
			witness, err := txscript.TaprootWitnessSignature(tx, sigHashes, i, utxo.Value, sourcePkScript, txscript.SigHashDefault, wif.PrivKey)
			if err != nil {
				return fmt.Errorf("failed to create taproot witness for input %d: %v", i, err)
			}
			txIn.Witness = witness
		default:
			return fmt.Errorf("unsupported address type for signing: %s", addressType)
		}
	}

	return nil
}

func p2wpkhRedeemScript(wif *btcutil.WIF) ([]byte, error) {
	pubKey := wif.PrivKey.PubKey()
	pubKeyHash := btcutil.Hash160(pubKey.SerializeCompressed())
	witnessAddr, err := btcutil.NewAddressWitnessPubKeyHash(pubKeyHash, &chaincfg.MainNetParams)
	if err != nil {
		return nil, err
	}
	return txscript.PayToAddrScript(witnessAddr)
}

func p2wpkhScriptCode(wif *btcutil.WIF) ([]byte, error) {
	pubKeyHash := btcutil.Hash160(wif.PrivKey.PubKey().SerializeCompressed())
	return txscript.NewScriptBuilder().
		AddOp(txscript.OP_DUP).
		AddOp(txscript.OP_HASH160).
		AddData(pubKeyHash).
		AddOp(txscript.OP_EQUALVERIFY).
		AddOp(txscript.OP_CHECKSIG).
		Script()
}

func createP2WPKHWitness(tx *wire.MsgTx, sigHashes *txscript.TxSigHashes, wif *btcutil.WIF, inputIndex int, utxo UTXO) error {
	script, err := p2wpkhScriptCode(wif)
	if err != nil {
		return fmt.Errorf("failed to create script: %v", err)
	}

	witness, err := txscript.WitnessSignature(tx, sigHashes, inputIndex, utxo.Value, script, txscript.SigHashAll, wif.PrivKey, true)
	if err != nil {
		return fmt.Errorf("failed to create witness signature: %v", err)
	}
	tx.TxIn[inputIndex].Witness = witness
	return nil
}

func (rpc *BitcoinRPC) broadcastTransaction(tx *wire.MsgTx) (string, error) {
	// REAL Bitcoin transaction broadcasting
	var buf bytes.Buffer
	if err := tx.Serialize(&buf); err != nil {
		return "", fmt.Errorf("failed to serialize transaction: %v", err)
	}

	// Encode to hex
	txHex := hex.EncodeToString(buf.Bytes())

	// Broadcast to Bitcoin network via API
	url := fmt.Sprintf("%s%s", rpc.api.BaseURL, rpc.api.BroadcastPath)

	var payload string
	var contentType string

	if rpc.api.BroadcastFormat == "raw" {
		// Raw hex in body (mempool.space)
		payload = txHex
		contentType = "text/plain"
	} else {
		// JSON format (blockstream.info, blockcypher)
		payload = fmt.Sprintf(`{"tx": "%s"}`, txHex)
		contentType = "application/json"
	}

	req, err := http.NewRequest("POST", url, strings.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("failed to create broadcast request: %v", err)
	}

	req.Header.Set("Content-Type", contentType)

	resp, err := rpc.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to broadcast transaction: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Read error response body
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("broadcast failed: %d - %s", resp.StatusCode, string(body))
	}

	// Return transaction hash
	return tx.TxHash().String(), nil
}
