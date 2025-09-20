package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"strings"

	eth "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// EthereumNetwork configuration for different Ethereum networks
type EthereumNetwork struct {
	Name    string
	RPC     string
	ChainID int64
}

// List of supported networks
var networks = []EthereumNetwork{
	{Name: "Ethereum", RPC: "https://eth-mainnet.g.alchemy.com/v2/demo", ChainID: 1},
	{Name: "Base", RPC: "https://mainnet.base.org", ChainID: 8453},
	{Name: "Optimism", RPC: "https://mainnet.optimism.io", ChainID: 10},
	{Name: "Arbitrum", RPC: "https://arb1.arbitrum.io/rpc", ChainID: 42161},
	{Name: "Polygon", RPC: "https://polygon-rpc.com", ChainID: 137},
	{Name: "BSC", RPC: "https://bsc-dataseed.binance.org", ChainID: 56},
	{Name: "Avalanche", RPC: "https://api.avax.network/ext/bc/C/rpc", ChainID: 43114},
}

// Major token addresses (USDC, LINK, etc.)
var majorTokens = map[string]string{
	// USDC addresses on different networks
	"0xA0b86a33E6441b6B4b1C2C2C2C2C2C2C2C2C2C2C": "USDC", // Ethereum USDC
	"0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913": "USDC", // Base USDC
	"0x0b2C639c533813f4Aa9D7837CAf62653d097Ff85": "USDC", // Optimism USDC
	"0xaf88d065e77c8cC2239327C5EDb3A432268e5831": "USDC", // Arbitrum USDC
	"0x2791Bca1f2de4661ED88A30C99A7a9449Aa84174": "USDC", // Polygon USDC
	"0x8AC76a51cc950d9822D68b83fE1Ad97B32Cd580d": "USDC", // BSC USDC
	"0xB97EF9Ef8734C71904D8002F8b6Bc66Dd9c48a6E": "USDC", // Avalanche USDC

	// USDT addresses
	"0xdAC17F958D2ee523a2206206994597C13D831ec7": "USDT", // Ethereum USDT
	"0x50c5725949A6F0c72E6C4a641F24049A917DB0Cb": "USDT", // Base USDT
	"0x94b008aA00579c1307B0EF2c499aD98a8ce58e58": "USDT", // Optimism USDT
	"0xFd086bC7CD5C481DCC9C85ebE478A1C0b69FCbb9": "USDT", // Arbitrum USDT
	"0xc2132D05D31c914a87C6611C10748AEb04B58e8F": "USDT", // Polygon USDT
	"0x55d398326f99059fF775485246999027B3197955": "USDT", // BSC USDT
	"0x9702230A8Ea53601f5cD2dc00fDBc13d4dF4A8c7": "USDT", // Avalanche USDT

	// DAI addresses
	"0x6B175474E89094C44Da98b954EedeAC495271d0F": "DAI", // Ethereum DAI
	"0xDA10009cBd5D07dd0CeCc66161FC93D7c9000da1": "DAI", // Base/Optimism/Arbitrum DAI (same address)
	"0x8f3Cf7ad23Cd3CaDbD9735AFf958023239c6A063": "DAI", // Polygon DAI
	"0x1AF3F329e8BE154074D8769D1FFa4eE058B1DBc3": "DAI", // BSC DAI
	"0xd586E7F844cEa2F87f50152665BCbc2C279D8d70": "DAI", // Avalanche DAI

	// BUSD addresses
	"0x4Fabb145d64652a948d72533023f6E7A623C7C53": "BUSD", // Ethereum BUSD
	"0xe9e7CEA3DedcA5984780Bafc599bD69ADd087D56": "BUSD", // BSC BUSD

	// FRAX addresses
	"0x853d955aCEf822Db058eb8505911ED77F175F99E": "FRAX", // Ethereum FRAX
	"0x2E3D870790dC77A83DD1d18184Acc7439A53f1ef": "FRAX", // Base FRAX
	"0x17FC002b466eEc40EaE3fBB9410285Bc5fA2212E": "FRAX", // Arbitrum FRAX
	"0x90C97F71E18723b0Cf0dfa30ee176Ab653E89F40": "FRAX", // BSC FRAX
	"0xD24C2D0964B8D3Dc0c9E997BFA7C7559f3D1aA60": "FRAX", // Avalanche FRAX

	// TUSD addresses
	"0x0000000000085d4780B73119b644AE5ecd22b376": "TUSD", // Ethereum TUSD
	"0x14016E85a25aeb13065688cAFB0B9c6b2C2C2C2C": "TUSD", // BSC TUSD

	// LINK addresses
	"0x514910771AF9Ca656af840dff83E8264EcF986CA": "LINK", // Ethereum LINK
	"0x779877A7B0D9E8603169DdbD7836e478b4624849": "LINK", // Base LINK
	"0x350a791Bfc2C21F9Ed5d10980Dad2e2638ffa7f6": "LINK", // Optimism LINK
	"0xf97f4df75117a78c1A5a0DBb814Af92458539FB4": "LINK", // Arbitrum LINK
	"0x53E0bca35eC356BD5ddDFebbd1Fc0fD03FaBad39": "LINK", // Polygon LINK
	"0xF8A0BF9cF54Bb92F17374d9e9A321E6a111a51bD": "LINK", // BSC LINK
	"0x5947BB275c521040051D82396192181b413227A3": "LINK", // Avalanche LINK

	// WETH addresses
	"0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2": "WETH", // Ethereum WETH
	"0x4200000000000000000000000000000000000006": "WETH", // Base/Optimism WETH (same address)
	"0x82aF49447D8a07e3bd95BD0d56f35241523fBab1": "WETH", // Arbitrum WETH
	"0x7ceB23fD6bC0adD59E62ac25578270cFf1b9f619": "WETH", // Polygon WETH
	"0x2170Ed0880ac9A755fd29B2688956BD959F933F8": "WETH", // BSC WETH
	"0x49D5c2BdFfac6CE2BFdB6640F4F80f226bc10bAB": "WETH", // Avalanche WETH
}

// ERC20 Transfer function signature
var transferFunctionSignature = crypto.Keccak256([]byte("transfer(address,uint256)"))[:4]

// EthereumBalance represents the balance information for an Ethereum address
type EthereumBalance struct {
	Address       string                       `json:"address"`
	ETHBalance    float64                      `json:"eth_balance"` // Balance in ETH
	TokenBalances map[string]EthereumTokenInfo `json:"token_balances"`
}

// EthereumTokenInfo represents token balance information
type EthereumTokenInfo struct {
	Symbol   string  `json:"symbol"`
	Balance  float64 `json:"balance"`
	Decimals int     `json:"decimals"`
}

// GetEthereumBalance retrieves the balance and token balances for an Ethereum address
func GetEthereumBalance(address string, networkName string, customRPC string) (*EthereumBalance, error) {
	// Determine RPC endpoint
	var rpcEndpoint string
	if customRPC != "" {
		rpcEndpoint = customRPC
	} else {
		// Find the network
		var network EthereumNetwork
		for _, n := range networks {
			if strings.EqualFold(n.Name, networkName) {
				network = n
				break
			}
		}
		if network.Name == "" {
			return nil, fmt.Errorf("unsupported network: %s", networkName)
		}
		rpcEndpoint = network.RPC
	}

	// Connect to the network
	client, err := ethclient.Dial(rpcEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RPC endpoint: %v", err)
	}
	defer client.Close()

	// Parse address
	addr := common.HexToAddress(address)

	// Check ETH balance
	balance, err := client.BalanceAt(context.Background(), addr, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance: %v", err)
	}

	// Convert balance from wei to ETH
	balanceEth := new(big.Float).Quo(new(big.Float).SetInt(balance), new(big.Float).SetInt(big.NewInt(1e18)))
	ethBalance, _ := balanceEth.Float64()

	// Get token balances by scanning for ERC20 tokens
	tokenBalances := make(map[string]EthereumTokenInfo)

	// Get all token transfers for this address to find tokens
	tokenAddresses, err := getTokenAddressesFromTransfers(client, addr)
	if err != nil {
		log.Printf("Warning: Could not get token addresses from transfers: %v", err)
		// Fallback to checking major tokens
		tokenAddresses = getMajorTokenAddresses()
	}

	for _, tokenAddr := range tokenAddresses {
		tokenContract := common.HexToAddress(tokenAddr)

		// Get token balance
		tokenBalance, decimals, err := getERC20BalanceAndDecimals(client, tokenContract, addr)
		if err != nil {
			continue // Skip if can't get balance
		}

		if tokenBalance.Cmp(big.NewInt(0)) > 0 {
			// Get token name and symbol from contract
			tokenName, _, err := getERC20TokenInfo(client, tokenContract)
			if err != nil {
				// Fallback to truncated address
				tokenName = tokenAddr[:8] + "..."
			}

			// Convert to float with proper decimals
			balanceFloat := new(big.Float).SetInt(tokenBalance)
			divisor := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil))
			balanceFloat.Quo(balanceFloat, divisor)
			balanceValue, _ := balanceFloat.Float64()

			tokenBalances[tokenAddr] = EthereumTokenInfo{
				Symbol:   tokenName,
				Balance:  balanceValue,
				Decimals: decimals,
			}
		}
	}

	return &EthereumBalance{
		Address:       address,
		ETHBalance:    ethBalance,
		TokenBalances: tokenBalances,
	}, nil
}

// SendEthereum sends ETH from one address to another
func SendEthereum(privateKeyHex, fromAddress, toAddress string, amount float64, networkName string, customRPC string) (string, error) {
	// Determine RPC endpoint and chain ID
	var rpcEndpoint string
	var chainID int64
	if customRPC != "" {
		rpcEndpoint = customRPC
		// For custom RPC, we'll need to get chain ID from the network
		// For now, default to mainnet chain ID
		chainID = 1
	} else {
		// Find the network
		var network EthereumNetwork
		for _, n := range networks {
			if strings.EqualFold(n.Name, networkName) {
				network = n
				break
			}
		}
		if network.Name == "" {
			return "", fmt.Errorf("unsupported network: %s", networkName)
		}
		rpcEndpoint = network.RPC
		chainID = chainID
	}

	// Parse the private key
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return "", fmt.Errorf("failed to parse private key: %v", err)
	}

	// Connect to the network
	client, err := ethclient.Dial(rpcEndpoint)
	if err != nil {
		return "", fmt.Errorf("failed to connect to RPC endpoint: %v", err)
	}
	defer client.Close()

	// Parse addresses
	fromAddr := common.HexToAddress(fromAddress)
	toAddr := common.HexToAddress(toAddress)

	// Check balance
	balance, err := client.BalanceAt(context.Background(), fromAddr, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get balance: %v", err)
	}

	// Convert amount to wei
	amountWei := new(big.Int)
	amountFloat := new(big.Float).SetFloat64(amount)
	amountFloat.Mul(amountFloat, new(big.Float).SetInt(big.NewInt(1e18)))
	amountFloat.Int(amountWei)

	if balance.Cmp(amountWei) < 0 {
		return "", fmt.Errorf("insufficient balance")
	}

	// Get gas price
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		return "", fmt.Errorf("failed to get gas price: %v", err)
	}

	// Use higher gas price for faster confirmation
	highGasPrice := new(big.Int).Mul(gasPrice, big.NewInt(2))

	// Estimate gas limit
	gasLimit := uint64(21000) // Standard ETH transfer

	// Get nonce
	nonce, err := client.PendingNonceAt(context.Background(), fromAddr)
	if err != nil {
		return "", fmt.Errorf("failed to get nonce: %v", err)
	}

	// Create transaction
	tx := types.NewTransaction(
		nonce,
		toAddr,
		amountWei,
		gasLimit,
		highGasPrice,
		nil, // data
	)

	// Sign transaction
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(big.NewInt(chainID)), privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign transaction: %v", err)
	}

	// Send transaction
	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return "", fmt.Errorf("failed to send transaction: %v", err)
	}

	log.Printf("Ethereum transaction sent - Amount: %f ETH, TXID: %s",
		amount, signedTx.Hash().Hex())

	return signedTx.Hash().Hex(), nil
}

// SendERC20Token sends ERC20 tokens from one address to another
func SendERC20Token(privateKeyHex, fromAddress, toAddress, tokenAddress string, amount float64, networkName string, customRPC string) (string, error) {
	// Determine RPC endpoint and chain ID
	var rpcEndpoint string
	var chainID int64
	if customRPC != "" {
		rpcEndpoint = customRPC
		// For custom RPC, we'll need to get chain ID from the network
		// For now, default to mainnet chain ID
		chainID = 1
	} else {
		// Find the network
		var network EthereumNetwork
		for _, n := range networks {
			if strings.EqualFold(n.Name, networkName) {
				network = n
				break
			}
		}
		if network.Name == "" {
			return "", fmt.Errorf("unsupported network: %s", networkName)
		}
		rpcEndpoint = network.RPC
		chainID = network.ChainID
	}

	// Parse the private key
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return "", fmt.Errorf("failed to parse private key: %v", err)
	}

	// Connect to the network
	client, err := ethclient.Dial(rpcEndpoint)
	if err != nil {
		return "", fmt.Errorf("failed to connect to RPC endpoint: %v", err)
	}
	defer client.Close()

	// Parse addresses
	fromAddr := common.HexToAddress(fromAddress)
	toAddr := common.HexToAddress(toAddress)
	tokenContract := common.HexToAddress(tokenAddress)

	// Get token balance and decimals
	tokenBalance, decimals, err := getERC20BalanceAndDecimals(client, tokenContract, fromAddr)
	if err != nil {
		return "", fmt.Errorf("failed to get token balance: %v", err)
	}

	// Convert amount to token units
	amountTokens := new(big.Int)
	amountFloat := new(big.Float).SetFloat64(amount)
	divisor := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil))
	amountFloat.Mul(amountFloat, divisor)
	amountFloat.Int(amountTokens)

	if tokenBalance.Cmp(amountTokens) < 0 {
		return "", fmt.Errorf("insufficient token balance")
	}

	// Get gas price
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		return "", fmt.Errorf("failed to get gas price: %v", err)
	}

	// Use higher gas price for faster confirmation
	highGasPrice := new(big.Int).Mul(gasPrice, big.NewInt(2))

	// ERC20 transfer function data
	transferData := append(transferFunctionSignature, toAddr.Bytes()...)
	amountBytes := amountTokens.Bytes()
	if len(amountBytes) < 32 {
		// Pad to 32 bytes
		padded := make([]byte, 32)
		copy(padded[32-len(amountBytes):], amountBytes)
		amountBytes = padded
	}
	transferData = append(transferData, amountBytes...)

	// Estimate gas limit
	gasLimit := uint64(100000) // Higher gas limit for token transfer

	// Get nonce
	nonce, err := client.PendingNonceAt(context.Background(), fromAddr)
	if err != nil {
		return "", fmt.Errorf("failed to get nonce: %v", err)
	}

	// Create transaction
	tx := types.NewTransaction(
		nonce,
		tokenContract,
		big.NewInt(0), // No ETH value for token transfer
		gasLimit,
		highGasPrice,
		transferData,
	)

	// Sign transaction
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(big.NewInt(chainID)), privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign transaction: %v", err)
	}

	// Send transaction
	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return "", fmt.Errorf("failed to send transaction: %v", err)
	}

	log.Printf("ERC20 token transaction sent - Amount: %f, TXID: %s",
		amount, signedTx.Hash().Hex())

	return signedTx.Hash().Hex(), nil
}

// Helper function to get ERC20 token balance and decimals
func getERC20BalanceAndDecimals(client *ethclient.Client, tokenContract, address common.Address) (*big.Int, int, error) {
	// ERC20 balanceOf function signature
	balanceOfSignature := crypto.Keccak256([]byte("balanceOf(address)"))[:4]
	// ERC20 decimals function signature
	decimalsSignature := crypto.Keccak256([]byte("decimals()"))[:4]

	// Get balance
	balanceData := append(balanceOfSignature, address.Bytes()...)
	balanceResult, err := client.CallContract(context.Background(), eth.CallMsg{
		To:   &tokenContract,
		Data: balanceData,
	}, nil)
	if err != nil {
		return big.NewInt(0), 0, err
	}

	// Get decimals
	decimalsResult, err := client.CallContract(context.Background(), eth.CallMsg{
		To:   &tokenContract,
		Data: decimalsSignature,
	}, nil)
	if err != nil {
		return big.NewInt(0), 0, err
	}

	// Parse the results
	var balance *big.Int
	var decimals int

	if len(balanceResult) >= 32 {
		balance = new(big.Int).SetBytes(balanceResult)
	} else {
		balance = big.NewInt(0)
	}

	if len(decimalsResult) >= 32 {
		decimals = int(new(big.Int).SetBytes(decimalsResult).Int64())
	} else {
		decimals = 18 // Default to 18 decimals
	}

	return balance, decimals, nil
}

// getTokenAddressesFromTransfers gets token addresses from transfer events
func getTokenAddressesFromTransfers(client *ethclient.Client, addr common.Address) ([]string, error) {
	// For now, return major token addresses as a fallback
	// In a full implementation, you would query transfer events to find all tokens
	return getMajorTokenAddresses(), nil
}

// getMajorTokenAddresses returns the list of major token addresses
func getMajorTokenAddresses() []string {
	var addresses []string
	for addr := range majorTokens {
		addresses = append(addresses, addr)
	}
	return addresses
}

// getERC20TokenInfo gets token name and symbol from the contract
func getERC20TokenInfo(client *ethclient.Client, tokenContract common.Address) (string, string, error) {
	// Function signatures for name() and symbol()
	nameSignature := crypto.Keccak256([]byte("name()"))[:4]
	symbolSignature := crypto.Keccak256([]byte("symbol()"))[:4]

	// Get name
	nameResult, err := client.CallContract(context.Background(), eth.CallMsg{
		To:   &tokenContract,
		Data: nameSignature,
	}, nil)
	if err != nil {
		return "", "", err
	}

	// Get symbol
	symbolResult, err := client.CallContract(context.Background(), eth.CallMsg{
		To:   &tokenContract,
		Data: symbolSignature,
	}, nil)
	if err != nil {
		return "", "", err
	}

	// Parse the results (they are returned as bytes32, need to extract the string)
	name := parseStringFromBytes(nameResult)
	symbol := parseStringFromBytes(symbolResult)

	return name, symbol, nil
}

// parseStringFromBytes extracts a string from contract call result
func parseStringFromBytes(data []byte) string {
	if len(data) < 32 {
		return ""
	}

	// Skip the first 32 bytes (offset) and get the length
	offset := int(new(big.Int).SetBytes(data[:32]).Int64())
	if offset >= len(data) {
		return ""
	}

	// Get the length of the string
	length := int(new(big.Int).SetBytes(data[offset : offset+32]).Int64())
	if length <= 0 || offset+32+length > len(data) {
		return ""
	}

	// Extract the string
	return string(data[offset+32 : offset+32+length])
}
