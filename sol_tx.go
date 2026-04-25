package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	sol "github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/programs/token"
	"github.com/gagliardetto/solana-go/rpc"
)

// Metaplex Token Metadata Program ID
var MetaplexTokenMetaProgramID = sol.MustPublicKeyFromBase58("metaqbxxUerdq28cj1RbAWkYQm3ybzjb6a8bt518x1s")

// SolanaBalance represents the balance information for a Solana address
type SolanaBalance struct {
	Address       string               `json:"address"`
	SOLBalance    float64              `json:"sol_balance"` // Balance in SOL
	TokenBalances map[string]TokenInfo `json:"token_balances"`
}

// TokenInfo represents token balance information
type TokenInfo struct {
	Symbol   string  `json:"symbol"`
	Balance  float64 `json:"balance"`
	Decimals int     `json:"decimals"`
	Mint     string  `json:"mint"`
}

// GetSolanaBalance retrieves the balance and token balances for a Solana address
func GetSolanaBalance(address string, customRPC string) (*SolanaBalance, error) {
	// Parse the address
	pubkey, err := sol.PublicKeyFromBase58(address)
	if err != nil {
		return nil, fmt.Errorf("invalid Solana address: %v", err)
	}

	// Determine RPC endpoint
	var rpcEndpoint string
	if customRPC != "" {
		rpcEndpoint = customRPC
	} else {
		rpcEndpoint = rpc.MainNetBeta_RPC
	}

	// Create RPC client
	client := rpc.New(rpcEndpoint)

	// Check SOL balance
	balance, err := client.GetBalance(context.Background(), pubkey, rpc.CommitmentFinalized)
	if err != nil {
		return nil, fmt.Errorf("failed to get SOL balance: %v", err)
	}

	// Convert lamports to SOL
	solBalance := float64(balance.Value) / float64(sol.LAMPORTS_PER_SOL)

	// Get token accounts
	tokenAccounts, err := client.GetTokenAccountsByOwner(
		context.Background(),
		pubkey,
		&rpc.GetTokenAccountsConfig{
			ProgramId: sol.TokenProgramID.ToPointer(),
		},
		&rpc.GetTokenAccountsOpts{
			Encoding: sol.EncodingBase64,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get token accounts: %v", err)
	}

	// Parse token balances
	tokenBalances := make(map[string]TokenInfo)
	for _, tokenAccount := range tokenAccounts.Value {
		accountData := tokenAccount.Account.Data.GetBinary()
		if len(accountData) < 72 {
			continue
		}

		// Extract amount (8 bytes starting at offset 64)
		amount := uint64(0)
		for i := 0; i < 8; i++ {
			amount |= uint64(accountData[64+i]) << (8 * i)
		}

		if amount > 0 {
			// Extract mint address (32 bytes starting at offset 0)
			mint := sol.PublicKeyFromBytes(accountData[0:32])

			// Get token metadata from RPC
			tokenInfo, err := getTokenMetadataFromRPC(client, mint.String())
			if err != nil {
				// Fallback to basic info if metadata fetch fails
				tokenInfo = TokenInfo{
					Symbol:   mint.String()[:8] + "...",
					Decimals: 9, // Default to 9 decimals
					Mint:     mint.String(),
				}
			}

			// Calculate balance with proper decimals
			decimals := tokenInfo.Decimals
			divisor := float64(1)
			for i := 0; i < decimals; i++ {
				divisor *= 10
			}
			balance := float64(amount) / divisor

			tokenBalances[mint.String()] = TokenInfo{
				Symbol:   tokenInfo.Symbol,
				Balance:  balance,
				Decimals: decimals,
				Mint:     mint.String(),
			}
		}
	}

	return &SolanaBalance{
		Address:       address,
		SOLBalance:    solBalance,
		TokenBalances: tokenBalances,
	}, nil
}

// SendSolana sends SOL from one address to another
func SendSolana(privateKeyBase58, fromAddress, toAddress string, amount string, customRPC string) (string, error) {
	// Parse the private key
	privateKey, err := sol.PrivateKeyFromBase58(privateKeyBase58)
	if err != nil {
		return "", fmt.Errorf("failed to parse private key: %v", err)
	}

	// Parse destination address
	destPubkey, err := sol.PublicKeyFromBase58(toAddress)
	if err != nil {
		return "", fmt.Errorf("invalid destination address: %v", err)
	}

	// Determine RPC endpoint
	var rpcEndpoint string
	if customRPC != "" {
		rpcEndpoint = customRPC
	} else {
		rpcEndpoint = rpc.MainNetBeta_RPC
	}

	// Create RPC client
	client := rpc.New(rpcEndpoint)

	// Check SOL balance
	balance, err := client.GetBalance(context.Background(), privateKey.PublicKey(), rpc.CommitmentFinalized)
	if err != nil {
		return "", fmt.Errorf("failed to get balance: %v", err)
	}

	// Convert amount to lamports
	amountLamports, err := parseAmountToUint64(amount, 9)
	if err != nil {
		return "", fmt.Errorf("invalid amount: %v", err)
	}

	if balance.Value < amountLamports {
		return "", fmt.Errorf("insufficient SOL balance")
	}

	// Create transfer instruction
	transferInstruction := system.NewTransferInstruction(
		amountLamports,
		privateKey.PublicKey(),
		destPubkey,
	)

	// Get recent blockhash
	recentBlockhash, err := client.GetLatestBlockhash(context.Background(), rpc.CommitmentFinalized)
	if err != nil {
		return "", fmt.Errorf("failed to get recent blockhash: %v", err)
	}

	// Create transaction
	tx, err := sol.NewTransaction(
		[]sol.Instruction{transferInstruction.Build()},
		recentBlockhash.Value.Blockhash,
		sol.TransactionPayer(privateKey.PublicKey()),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create transaction: %v", err)
	}

	// Sign transaction
	_, err = tx.Sign(func(key sol.PublicKey) *sol.PrivateKey {
		if key.Equals(privateKey.PublicKey()) {
			return &privateKey
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("failed to sign transaction: %v", err)
	}

	// Send transaction
	sig, err := client.SendTransaction(context.Background(), tx)
	if err != nil {
		return "", fmt.Errorf("failed to send transaction: %v", err)
	}

	log.Printf("Solana transaction sent - Amount: %s SOL, TXID: %s", amount, sig.String())
	return sig.String(), nil
}

// SendSPLToken sends SPL tokens from one address to another
func SendSPLToken(privateKeyBase58, fromAddress, toAddress, tokenMint string, amount string, customRPC string) (string, error) {
	// Parse the private key
	privateKey, err := sol.PrivateKeyFromBase58(privateKeyBase58)
	if err != nil {
		return "", fmt.Errorf("failed to parse private key: %v", err)
	}

	// Parse destination address
	destPubkey, err := sol.PublicKeyFromBase58(toAddress)
	if err != nil {
		return "", fmt.Errorf("invalid destination address: %v", err)
	}

	// Parse token mint
	mint, err := sol.PublicKeyFromBase58(tokenMint)
	if err != nil {
		return "", fmt.Errorf("invalid token mint: %v", err)
	}

	// Determine RPC endpoint
	var rpcEndpoint string
	if customRPC != "" {
		rpcEndpoint = customRPC
	} else {
		rpcEndpoint = rpc.MainNetBeta_RPC
	}

	// Create RPC client
	client := rpc.New(rpcEndpoint)

	// Get source token account
	sourceTokenAccount, _, err := sol.FindAssociatedTokenAddress(privateKey.PublicKey(), mint)
	if err != nil {
		return "", fmt.Errorf("failed to find source token account: %v", err)
	}

	// Get destination token account
	destTokenAccount, _, err := sol.FindAssociatedTokenAddress(destPubkey, mint)
	if err != nil {
		return "", fmt.Errorf("failed to find destination token account: %v", err)
	}

	// Get token account info to check balance and decimals
	sourceAccountInfo, err := client.GetAccountInfo(context.Background(), sourceTokenAccount)
	if err != nil {
		return "", fmt.Errorf("failed to get source token account info: %v", err)
	}

	if sourceAccountInfo.Value == nil {
		return "", fmt.Errorf("source token account not found")
	}

	// Parse account data to get balance
	accountData := sourceAccountInfo.Value.Data.GetBinary()
	if len(accountData) < 72 {
		return "", fmt.Errorf("invalid token account data")
	}

	// Extract current balance
	currentAmount := uint64(0)
	for i := 0; i < 8; i++ {
		currentAmount |= uint64(accountData[64+i]) << (8 * i)
	}

	tokenInfo, err := getTokenMetadataFromRPC(client, tokenMint)
	if err != nil {
		return "", fmt.Errorf("failed to get token mint decimals: %v", err)
	}

	amountTokens, err := parseAmountToUint64(amount, tokenInfo.Decimals)
	if err != nil {
		return "", fmt.Errorf("invalid amount: %v", err)
	}

	if currentAmount < amountTokens {
		return "", fmt.Errorf("insufficient token balance")
	}

	// Create token transfer instruction
	tokenTransferInstruction := token.NewTransferInstruction(
		amountTokens,
		sourceTokenAccount,
		destTokenAccount,
		privateKey.PublicKey(),
		[]sol.PublicKey{},
	)

	// Get recent blockhash
	recentBlockhash, err := client.GetLatestBlockhash(context.Background(), rpc.CommitmentFinalized)
	if err != nil {
		return "", fmt.Errorf("failed to get recent blockhash: %v", err)
	}

	// Create transaction
	tx, err := sol.NewTransaction(
		[]sol.Instruction{tokenTransferInstruction.Build()},
		recentBlockhash.Value.Blockhash,
		sol.TransactionPayer(privateKey.PublicKey()),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create transaction: %v", err)
	}

	// Sign transaction
	_, err = tx.Sign(func(key sol.PublicKey) *sol.PrivateKey {
		if key.Equals(privateKey.PublicKey()) {
			return &privateKey
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("failed to sign transaction: %v", err)
	}

	// Send transaction
	sig, err := client.SendTransaction(context.Background(), tx)
	if err != nil {
		return "", fmt.Errorf("failed to send transaction: %v", err)
	}

	log.Printf("SPL token transaction sent - Amount: %s, Mint: %s, TXID: %s", amount, tokenMint, sig.String())
	return sig.String(), nil
}

// getTokenMetadataFromRPC retrieves token metadata from Solana RPC
func getTokenMetadataFromRPC(client *rpc.Client, mint string) (TokenInfo, error) {
	// Parse mint address
	mintPubkey, err := sol.PublicKeyFromBase58(mint)
	if err != nil {
		return TokenInfo{}, fmt.Errorf("invalid mint address: %v", err)
	}

	// Get mint account info to get decimals
	mintAccountInfo, err := client.GetAccountInfo(context.Background(), mintPubkey)
	if err != nil {
		return TokenInfo{}, fmt.Errorf("failed to get mint account info: %v", err)
	}

	if mintAccountInfo.Value == nil {
		return TokenInfo{}, fmt.Errorf("mint account not found")
	}

	// Parse mint account data to get decimals
	mintData := mintAccountInfo.Value.Data.GetBinary()
	if len(mintData) < 44 {
		return TokenInfo{}, fmt.Errorf("invalid mint account data")
	}

	// Decimals are at offset 44 in mint account data
	decimals := int(mintData[44])

	// Try to get metadata from Metaplex metadata program
	metadataPubkey, _, err := sol.FindProgramAddress([][]byte{
		[]byte("metadata"),
		MetaplexTokenMetaProgramID.Bytes(),
		mintPubkey.Bytes(),
	}, MetaplexTokenMetaProgramID)
	if err != nil {
		return TokenInfo{
			Symbol:   mint[:8] + "...",
			Decimals: decimals,
			Mint:     mint,
		}, nil
	}

	// Get metadata account
	metadataAccountInfo, err := client.GetAccountInfo(context.Background(), metadataPubkey)
	if err != nil || metadataAccountInfo.Value == nil {
		// No metadata found, return basic info5EiSjMaqkaG6ATXy42bbUR1zT1kGGq9ZqQLXTirc6Q4ZiY8fPAEfBvjKy7tJ9LoQu9U1AktdoAvQ2oGTdptFMoyX
		return TokenInfo{
			Symbol:   mint[:8] + "...",
			Decimals: decimals,
			Mint:     mint,
		}, nil
	}

	// Parse metadata to extract symbol
	metadataData := metadataAccountInfo.Value.Data.GetBinary()
	if len(metadataData) < 4 {
		return TokenInfo{
			Symbol:   mint[:8] + "...",
			Decimals: decimals,
			Mint:     mint,
		}, nil
	}

	// Skip the first 4 bytes (discriminator)
	metadataData = metadataData[4:]

	// Try to extract symbol from metadata
	symbol := extractSymbolFromMetadata(metadataData)
	if symbol == "" {
		// Try alternative approach - look for any readable text in the metadata
		symbol = extractAnyReadableText(metadataData)
	}
	if symbol == "" {
		// Try to get token info from Jupiter API as fallback
		symbol = getTokenNameFromJupiter(mint)
	}
	if symbol == "" {
		symbol = mint[:8] + "..."
	}

	return TokenInfo{
		Symbol:   symbol,
		Decimals: decimals,
		Mint:     mint,
	}, nil
}

// extractSymbolFromMetadata extracts the symbol from Metaplex metadata
func extractSymbolFromMetadata(data []byte) string {
	// Convert to string for easier parsing
	dataStr := string(data)

	// Look for "name" field first (more reliable than symbol)
	name := extractFieldFromMetadata(dataStr, "name")
	if name != "" {
		return name
	}

	// Fallback to symbol field
	symbol := extractFieldFromMetadata(dataStr, "symbol")
	if symbol != "" {
		return symbol
	}

	return ""
}

// extractFieldFromMetadata extracts a specific field from metadata
func extractFieldFromMetadata(dataStr, field string) string {
	// Try different patterns that might appear in the metadata
	patterns := []string{
		`"` + field + `":"`,
		`"` + field + `": "`,
		`"` + field + `":`,
		field + `":"`,
		field + `": "`,
	}

	for _, pattern := range patterns {
		if idx := findPattern(dataStr, pattern); idx != -1 {
			// Found the pattern, extract the value
			start := idx + len(pattern)
			end := findStringEnd(dataStr, start)
			if end != -1 {
				value := dataStr[start:end]
				// Clean up the value
				value = strings.Trim(value, `"`)
				if value != "" {
					return value
				}
			}
		}
	}

	return ""
}

// Helper functions for parsing metadata
func findPattern(data, pattern string) int {
	for i := 0; i <= len(data)-len(pattern); i++ {
		if data[i:i+len(pattern)] == pattern {
			return i
		}
	}
	return -1
}

func findStringEnd(data string, start int) int {
	for i := start; i < len(data); i++ {
		if data[i] == '"' && (i == start || data[i-1] != '\\') {
			return i
		}
	}
	return -1
}

// extractAnyReadableText tries to find any readable text in the metadata
func extractAnyReadableText(data []byte) string {
	dataStr := string(data)

	// Look for common token name patterns
	// First, try to find "name" or "symbol" followed by readable text
	patterns := []string{
		"name\":\"",
		"symbol\":\"",
		"name\": \"",
		"symbol\": \"",
	}

	for _, pattern := range patterns {
		if idx := strings.Index(dataStr, pattern); idx != -1 {
			start := idx + len(pattern)
			end := findStringEnd(dataStr, start)
			if end != -1 {
				name := dataStr[start:end]
				name = strings.Trim(name, `"`)
				if len(name) > 0 && len(name) < 50 {
					return name
				}
			}
		}
	}

	// Look for readable text sequences (letters, numbers, dots, spaces, parentheses)
	for i := 0; i < len(dataStr)-2; i++ {
		if isReadableChar(dataStr[i]) {
			// Found start of potential name
			name := ""
			for j := i; j < len(dataStr) && j < i+50; j++ {
				if isReadableChar(dataStr[j]) {
					name += string(dataStr[j])
				} else {
					break
				}
			}
			// Return the longest readable sequence that looks like a token name
			if len(name) >= 3 && len(name) <= 50 && looksLikeTokenName(name) {
				return name
			}
		}
	}

	return ""
}

// isReadableChar checks if a character is part of a readable token name
func isReadableChar(c byte) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') ||
		(c >= '0' && c <= '9') || c == '.' || c == ' ' ||
		c == '(' || c == ')' || c == '#' || c == '-' || c == '_'
}

// looksLikeTokenName checks if a string looks like a token name
func looksLikeTokenName(s string) bool {
	// Must contain at least one letter
	hasLetter := false
	for _, c := range s {
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') {
			hasLetter = true
			break
		}
	}
	return hasLetter && !strings.Contains(s, "http") && !strings.Contains(s, "ipfs")
}

// getTokenNameFromJupiter tries to get token info from Jupiter API
func getTokenNameFromJupiter(mint string) string {
	// Jupiter API endpoint for token list
	url := "https://token.jup.ag/strict"

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	var tokens []struct {
		Address string `json:"address"`
		Symbol  string `json:"symbol"`
		Name    string `json:"name"`
	}

	if err := json.Unmarshal(body, &tokens); err != nil {
		return ""
	}

	// Look for the token by mint address
	for _, token := range tokens {
		if token.Address == mint {
			if token.Name != "" {
				return token.Name
			}
			if token.Symbol != "" {
				return token.Symbol
			}
		}
	}

	return ""
}
