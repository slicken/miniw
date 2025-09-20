package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
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
      --prefix             Addon for include (Optional for key-pair gen.)
      --postfix            Addon for include (Optional for key-pair gen.)
                           Example: -i abcde,10000
  --custom_private         Use custom private key
  --custom_mnemonic        Use custom mnemonic
  --custom_path            Use custom derivation path (Optional)

Action Commands (Mnemonic or Private Key required)
  --balance                Check balance for generated wallet
  --send                   Send cryptocurrency
    --amount <amount>      Amount to send
    --to <address>         Destination address
      --token <address>    Custom Token contract address (Optional)
  --rpc <url>              Custom RPC endpoint for the chosen network (Optional)

Examples:
  %s btc --custom_private=5KJvsngHeMpm884wtkJHQtFvi... --balance
  %s eth --custom_private=ddcc8e6a9be77249cb44a7d3b... --balance --rpc=ETH_L2_RPC
  %s sol --custom_mnemonic="abandon abandon abandon..." --send --amount=0.1 --to=8TinVypdVXQcLoTkr2ezbVumquEoWpt...
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
	customMnemonicFlag = flag.String("custom_mnemonic", "", "Custom mnemonic phrase for key generation.")
	customPathFlag     = flag.String("custom_path", "", "Custom derivation path for key generation.")
	customPrivateFlag  = flag.String("custom_private", "", "Custom private key for key generation.")

	// Action command flags
	balanceFlag = flag.Bool("balance", false, "Check balance for generated wallet.")
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

	// Check if balance or send commands are requested
	if *balanceFlag || *sendFlag {
		// Validate that mnemonic or private key is provided for action commands
		if customMnemonic == "" && customPrivate == "" {
			log.Fatal("Error: Action commands (--balance, --send) require either --custom_mnemonic or --custom_private to be provided.")
		}
	}

	if *balanceFlag {
		handleBalanceCommand(keyPair, networkArg)
		return
	}

	if *sendFlag {
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

	// For the include/vanity address generation loop
	count := 0
	for {
		keyPair, err := network.GenerateKeys()
		if err != nil {
			fmt.Println(networkArg, err)
			return
		}

		for _, word := range includeWords {
			if *preFlag {
				if strings.EqualFold(keyPair.public[:len(word)], word) {
					fmt.Printf("                 %s included as prefix in public key below\n", word)
					keyPair.Print()
					fmt.Println("")
					count++
					break
				}
			}
			if *postFlag {
				if strings.EqualFold(keyPair.public[len(keyPair.public)-len(word):], word) {
					fmt.Printf("                 %s included as postfix in public key below\n", word)
					keyPair.Print()
					fmt.Println("")
					count++
					break
				}
			}
			if !*preFlag && !*postFlag {
				for i := 0; i < len(keyPair.public)-len(word)+1; i++ {
					if strings.EqualFold(keyPair.public[i:i+len(word)], word) {
						fmt.Printf("                 %s included in public key below\n", word)
						keyPair.Print()
						fmt.Println("")
						count++
						break
					}
				}
			}
		}
		if count > 10 {
			break
		}
	}
}

// handleBalanceCommand processes balance checking using generated keypair
func handleBalanceCommand(keyPair *KeyPair, networkArg string) {
	// Determine network type and call appropriate function
	switch strings.ToLower(networkArg) {
	case "btc", "legacy", "bitcoin":
		balance, err := GetBitcoinBalance(keyPair.public, "legacy", *rpcFlag)
		if err != nil {
			log.Fatalf("Failed to get Bitcoin balance: %v", err)
		}
		printBitcoinBalance(balance)

	case "btcs", "segwit":
		balance, err := GetBitcoinBalance(keyPair.public, "segwit", *rpcFlag)
		if err != nil {
			log.Fatalf("Failed to get Bitcoin SegWit balance: %v", err)
		}
		printBitcoinBalance(balance)

	case "btcn", "native":
		balance, err := GetBitcoinBalance(keyPair.public, "native_segwit", *rpcFlag)
		if err != nil {
			log.Fatalf("Failed to get Bitcoin Native SegWit balance: %v", err)
		}
		printBitcoinBalance(balance)

	case "btct", "taproot":
		balance, err := GetBitcoinBalance(keyPair.public, "taproot", *rpcFlag)
		if err != nil {
			log.Fatalf("Failed to get Bitcoin Taproot balance: %v", err)
		}
		printBitcoinBalance(balance)

	case "eth", "ethereum":
		balance, err := GetEthereumBalance(keyPair.public, "ethereum", *rpcFlag)
		if err != nil {
			log.Fatalf("Failed to get Ethereum balance: %v", err)
		}
		printEthereumBalance(balance)

	case "sol", "solana":
		balance, err := GetSolanaBalance(keyPair.public, *rpcFlag)
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
