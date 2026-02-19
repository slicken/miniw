package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

type KeyPair struct {
	network        string
	public         string
	private        string
	mnemonic       string
	derivationPath string
}

func Usage() {
	fmt.Printf(`Usage: %s <NETWORK> | [WALLET] | <ACTION>

Network (required as first argument):
  btc, legacy              Legacy (P2PKH): Oldest type, less efficient, higher fees.
  btcn, native             Native SegWit (P2WPKH, Bech32): More efficient and secure, lower fees.
  btcs, segwit             SegWit (P2SH-wrapped P2WPKH): SegWit compatibility, lower fees.
  btct, taproot            Taproot (P2TR): Latest Bitcoin upgrade, more privacy and efficiency.
  eth, ethereum            Ethereum
  sol, solana              Solana

Wallet Options:
  -a, --all                Prints mnemonic and derivation path
  -i, --include <include>  Include words in public key (comma-separated)
      --prefix             Addon for include (optional for key-pair gen.)
      --postfix            Addon for include (optional for key-pair gen.)
                           Example: -i abcde,10000
  --privatekey             Use custom private key
  --mnemonic               Use custom mnemonic
  --path                   Use custom derivation path (optional)

Action Commands:
  --balance <wallet>       Check balance for wallet address
  --send                   Send cryptocurrency (requires --privatekey or --mnemonic)
    --amount <amount>      Amount to send
    --to <address>         Destination address
      --token <address>    Custom Token contract address (optional)
  --rpc <url>              Custom RPC endpoint for the chosen network (optional)

Examples:
  %s btc --balance=1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa
  %s eth --balance=0x742d35Cc6634C0532925a3b8D4C9db96C4b4d8b6 --rpc=ETH_L2_RPC
  %s sol --mnemonic="abandon abandon abandon..." --send --amount=0.1 --to=8TinVypdVXQcLoTkr2ezbVumquEoWpt...
`, os.Args[0], os.Args[0], os.Args[0], os.Args[0])
	os.Exit(1)
}

// Print to std.out
func (k KeyPair) Print() {
	fmt.Printf("%-3s %-12s %s\n", k.network, "public", k.public)
	fmt.Printf("%-3s %-12s %s\n", k.network, "private", k.private)
	if k.mnemonic != "" {
		fmt.Printf("%-3s %-12s %s\n", k.network, "mnemonic", k.mnemonic)
		if k.derivationPath != "" {
			fmt.Printf("%-3s %-12s %s\n", k.network, "derivation", k.derivationPath)
		}
	}
}

type Network interface {
	Name() string
	GenerateKeys() (*KeyPair, error)
}

var (
	includeFlag        = flag.String("i", "", "A comma-separated list of characters or words that the public key should include.")
	includeLongFlag    = flag.String("include", "", "A comma-separated list of characters or words that the public key should include.")
	preFlag            = flag.Bool("prefix", false, "Addon to include flag.")
	postFlag           = flag.Bool("postfix", false, "Addon to include flag.")
	infoFlag           = flag.Bool("a", false, "A boolean flag to generate and print a mnemonic.")
	infoLongFlag       = flag.Bool("all", false, "A boolean flag to generate and print a mnemonic.")
	customMnemonicFlag = flag.String("mnemonic", "", "Custom mnemonic phrase for key generation.")
	customPathFlag     = flag.String("path", "", "Custom derivation path for key generation.")
	customPrivateFlag  = flag.String("privatekey", "", "Custom private key for key generation.")

	// Action command flags
	balanceFlag = flag.String("balance", "", "Check balance for wallet address.")
	sendFlag    = flag.Bool("send", false, "Send cryptocurrency.")
	amountFlag  = flag.String("amount", "", "Amount to send.")
	toFlag      = flag.String("to", "", "Destination address.")
	tokenFlag   = flag.String("token", "", "Token contract address for ERC20 tokens.")
	rpcFlag     = flag.String("rpc", "", "Custom RPC endpoint for the chosen network.")

	customMnemonic string
	customPath     string
	customPrivate  string
)

func main() {
	flag.Usage = Usage

	// Manually parse the arguments to separate the network argument from the flags
	args := os.Args[1:]
	var networkArg string
	var flagArgs []string

	for _, arg := range args {
		if !strings.HasPrefix(arg, "-") && networkArg == "" {
			networkArg = arg // Assume the first non-flag argument is the network
		} else {
			flagArgs = append(flagArgs, arg)
		}
	}

	// Validate network argument
	if networkArg == "" {
		Usage()
	}

	// Rebuild os.Args for flags parsing
	os.Args = append([]string{os.Args[0]}, flagArgs...)
	flag.Parse()

	// Assign custom values
	customMnemonic = *customMnemonicFlag
	customPath = *customPathFlag
	customPrivate = *customPrivateFlag

	// Proceed with the rest of the program
	var network Network
	switch strings.ToLower(networkArg) {
	case "btc", "legacy", "bitcoin":
		network = btcMap["legacy"]
	case "btcs", "segwit":
		network = btcMap["segwit"]
	case "btcn", "native":
		network = btcMap["native"]
	case "btct", "taproot":
		network = btcMap["taproot"]
	case "eth", "ethereum":
		network = &ethereum{}
	case "sol", "solana":
		network = &solana{}
	default:
		log.Fatalf("%q not found\n", networkArg)
	}

	// Generate the keypair using existing logic
	keyPair, err := network.GenerateKeys()
	if err != nil {
		log.Fatalln(networkArg, err)
	}

	// Check if balance command is requested
	if *balanceFlag != "" {
		handleBalanceCommand(*balanceFlag, networkArg)
		return
	}

	// Check if send command is requested
	if *sendFlag {
		// Validate that mnemonic or private key is provided for send command
		if customMnemonic == "" && customPrivate == "" {
			log.Fatal("Error: --send requires either --mnemonic or --privatekey to be provided.")
		}
		handleSendCommand(keyPair, networkArg)
		return
	}

	// Original key generation logic for include/vanity addresses
	include := *includeFlag
	if *includeLongFlag != "" {
		include = *includeLongFlag
	}

	includeWords := strings.Split(include, ",")
	if include != "" && len(includeWords) == 0 {
		log.Fatalln("no words to include")
	}

	// If we just want to generate a keypair without include logic
	if include == "" {
		keyPair.Print()
		return
	}

	// For the include/vanity address generation loop - use all CPU cores
	numCPUs := runtime.NumCPU()
	runtime.GOMAXPROCS(numCPUs)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	resultsCh := make(chan *KeyPair, numCPUs*2)
	errCh := make(chan error, 1)

	var wg sync.WaitGroup
	searchMode := "in"
	if *preFlag && *postFlag {
		searchMode = "as prefix or postfix in"
	} else if *preFlag {
		searchMode = "as prefix in"
	} else if *postFlag {
		searchMode = "as postfix in"
	}
	fmt.Printf("Searching for [%s] %s public key (using %d CPU cores)...\n\n", strings.Join(includeWords, ", "), searchMode, numCPUs)

	for i := 0; i < numCPUs; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				default:
					keyPair, err := network.GenerateKeys()
					if err != nil {
						select {
						case errCh <- err:
						default:
						}
						return
					}

					for _, word := range includeWords {
						if *preFlag {
							if len(keyPair.public) >= len(word) && strings.EqualFold(keyPair.public[:len(word)], word) {
								select {
								case resultsCh <- keyPair:
								case <-ctx.Done():
									return
								}
								break
							}
						}
						if *postFlag {
							if len(keyPair.public) >= len(word) && strings.EqualFold(keyPair.public[len(keyPair.public)-len(word):], word) {
								select {
								case resultsCh <- keyPair:
								case <-ctx.Done():
									return
								}
								break
							}
						}
						if !*preFlag && !*postFlag {
							for j := 0; j < len(keyPair.public)-len(word)+1; j++ {
								if strings.EqualFold(keyPair.public[j:j+len(word)], word) {
									select {
									case resultsCh <- keyPair:
									case <-ctx.Done():
										return
									}
									break
								}
							}
						}
					}
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	count := int32(0)
	for {
		select {
		case err := <-errCh:
			cancel()
			fmt.Println(networkArg, err)
			return
		case kp, ok := <-resultsCh:
			if !ok {
				return
			}
			kp.Print()
			fmt.Println("")
			if atomic.AddInt32(&count, 1) > 10 {
				cancel()
				return
			}
		}
	}
}

// handleBalanceCommand processes balance checking using wallet address
func handleBalanceCommand(walletAddress string, networkArg string) {
	// Validate wallet address is provided
	if walletAddress == "" {
		log.Fatalln("Error: wallet address is missing. Please provide a wallet address after --balance.")
	}

	// Determine network type and call appropriate function
	switch strings.ToLower(networkArg) {
	case "btc", "legacy", "bitcoin":
		balance, err := GetBitcoinBalance(walletAddress, "legacy", *rpcFlag)
		if err != nil {
			log.Fatalf("Failed to get Bitcoin balance: %v", err)
		}
		printBitcoinBalance(balance)

	case "btcs", "segwit":
		balance, err := GetBitcoinBalance(walletAddress, "segwit", *rpcFlag)
		if err != nil {
			log.Fatalf("Failed to get Bitcoin SegWit balance: %v", err)
		}
		printBitcoinBalance(balance)

	case "btcn", "native":
		balance, err := GetBitcoinBalance(walletAddress, "native_segwit", *rpcFlag)
		if err != nil {
			log.Fatalf("Failed to get Bitcoin Native SegWit balance: %v", err)
		}
		printBitcoinBalance(balance)

	case "btct", "taproot":
		balance, err := GetBitcoinBalance(walletAddress, "taproot", *rpcFlag)
		if err != nil {
			log.Fatalf("Failed to get Bitcoin Taproot balance: %v", err)
		}
		printBitcoinBalance(balance)

	case "eth", "ethereum":
		balance, err := GetEthereumBalance(walletAddress, "ethereum", *rpcFlag)
		if err != nil {
			log.Fatalf("Failed to get Ethereum balance: %v", err)
		}
		printEthereumBalance(balance)

	case "sol", "solana":
		balance, err := GetSolanaBalance(walletAddress, *rpcFlag)
		if err != nil {
			log.Fatalf("Failed to get Solana balance: %v", err)
		}
		printSolanaBalance(balance)

	default:
		log.Fatalf("Unsupported network: %s", networkArg)
	}
}

// handleSendCommand processes sending transactions using generated keypair
func handleSendCommand(keyPair *KeyPair, networkArg string) {
	amount := *amountFlag
	to := *toFlag
	token := *tokenFlag

	if amount == "" {
		log.Fatal("Amount is required for send command. Use --amount flag.")
	}
	if to == "" {
		log.Fatal("Destination address is required for send command. Use --to flag.")
	}

	// Parse amount
	amountFloat, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		log.Fatalf("Invalid amount: %v", err)
	}

	// Determine network type and call appropriate function
	switch strings.ToLower(networkArg) {
	case "btc", "legacy", "bitcoin":
		txHash, err := SendBitcoin(keyPair.private, keyPair.public, to, amountFloat, "legacy", *rpcFlag)
		if err != nil {
			log.Fatalf("Failed to send Bitcoin: %v", err)
		}
		fmt.Printf("Bitcoin transaction sent successfully. TXID: %s\n", txHash)

	case "btcs", "segwit":
		txHash, err := SendBitcoin(keyPair.private, keyPair.public, to, amountFloat, "segwit", *rpcFlag)
		if err != nil {
			log.Fatalf("Failed to send Bitcoin SegWit: %v", err)
		}
		fmt.Printf("Bitcoin SegWit transaction sent successfully. TXID: %s\n", txHash)

	case "btcn", "native":
		txHash, err := SendBitcoin(keyPair.private, keyPair.public, to, amountFloat, "native_segwit", *rpcFlag)
		if err != nil {
			log.Fatalf("Failed to send Bitcoin Native SegWit: %v", err)
		}
		fmt.Printf("Bitcoin Native SegWit transaction sent successfully. TXID: %s\n", txHash)

	case "btct", "taproot":
		txHash, err := SendBitcoin(keyPair.private, keyPair.public, to, amountFloat, "taproot", *rpcFlag)
		if err != nil {
			log.Fatalf("Failed to send Bitcoin Taproot: %v", err)
		}
		fmt.Printf("Bitcoin Taproot transaction sent successfully. TXID: %s\n", txHash)

	case "eth", "ethereum":
		if token != "" {
			// Send ERC20 token
			txHash, err := SendERC20Token(keyPair.private, keyPair.public, to, token, amountFloat, "ethereum", *rpcFlag)
			if err != nil {
				log.Fatalf("Failed to send ERC20 token: %v", err)
			}
			fmt.Printf("ERC20 token transaction sent successfully. TXID: %s\n", txHash)
		} else {
			// Send ETH
			txHash, err := SendEthereum(keyPair.private, keyPair.public, to, amountFloat, "ethereum", *rpcFlag)
			if err != nil {
				log.Fatalf("Failed to send Ethereum: %v", err)
			}
			fmt.Printf("Ethereum transaction sent successfully. TXID: %s\n", txHash)
		}

	case "sol", "solana":
		if token != "" {
			// Send SPL token
			txHash, err := SendSPLToken(keyPair.private, keyPair.public, to, token, amountFloat, *rpcFlag)
			if err != nil {
				log.Fatalf("Failed to send SPL token: %v", err)
			}
			fmt.Printf("SPL token transaction sent successfully. TXID: %s\n", txHash)
		} else {
			// Send SOL
			txHash, err := SendSolana(keyPair.private, keyPair.public, to, amountFloat, *rpcFlag)
			if err != nil {
				log.Fatalf("Failed to send Solana: %v", err)
			}
			fmt.Printf("Solana transaction sent successfully. TXID: %s\n", txHash)
		}

	default:
		log.Fatalf("Unsupported network: %s", networkArg)
	}
}

// printBitcoinBalance prints Bitcoin balance information
func printBitcoinBalance(balance *BitcoinBalance) {
	fmt.Printf("Bitcoin Balance for %s:\n", balance.Address)
	fmt.Printf("  Balance: %.8f BTC\n", balance.Balance)
	fmt.Printf("  UTXOs: %d\n", len(balance.UTXOs))

	if len(balance.UTXOs) > 0 {
		fmt.Println("  UTXO Details:")
		for i, utxo := range balance.UTXOs {
			if i >= 5 { // Limit to first 5 UTXOs
				fmt.Printf("    ... and %d more UTXOs\n", len(balance.UTXOs)-5)
				break
			}
			fmt.Printf("    TXID: %s, Vout: %d, Value: %.8f BTC\n",
				utxo.TxID, utxo.Vout, float64(utxo.Value)/100000000)
		}
	}
}

// printEthereumBalance prints Ethereum balance information
func printEthereumBalance(balance *EthereumBalance) {
	fmt.Printf("Ethereum Balance for %s:\n", balance.Address)
	fmt.Printf("  ETH Balance: %.6f ETH\n", balance.ETHBalance)

	if len(balance.TokenBalances) > 0 {
		fmt.Println("  Token Balances:")
		for tokenAddr, tokenInfo := range balance.TokenBalances {
			fmt.Printf("    %s (%s): %.6f\n", tokenInfo.Symbol, tokenAddr[:10]+"...", tokenInfo.Balance)
		}
	}
}

// printSolanaBalance prints Solana balance information
func printSolanaBalance(balance *SolanaBalance) {
	fmt.Printf("Solana Balance for %s:\n", balance.Address)
	fmt.Printf("  SOL Balance: %.6f SOL\n", balance.SOLBalance)

	if len(balance.TokenBalances) > 0 {
		fmt.Println("  Token Balances:")
		for mint, tokenInfo := range balance.TokenBalances {
			fmt.Printf("    %12.6f  %-12s  %s\n", tokenInfo.Balance, tokenInfo.Symbol, mint)
		}
	}
}
