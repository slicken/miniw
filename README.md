# mmcw - mini multi crypto wallet

**mmcw** is a simple but powerful lightwaight crypto multi wallet and key generation application that generates self-custodial wallet key pairs and manages cryptocurrency transactions for the following blockchains:

- Bitcoin (legacy, SegWit, Native SegWit, Taproot)
- Solana
- Ethereum

## Features

- **Key Generation**: Generate wallet key pairs for multiple blockchain networks
- **Balance Checking**: Check wallet balances and token holdings
- **Transaction Sending**: Send native currencies and tokens
- **Custom RPC**: Use custom RPC endpoints for any network
- **Vanity Addresses**: Generate addresses with specific patterns

## Usage

### Key Generation
```bash
$ ./mmcw
Usage: ./mmwc <NETWORK> | [WALLET] | <ACTION>

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
  ./generateKeys btc --custom_private=5KJvsngHeMpm884wtkJHQtFvi... --balance
  ./generateKeys eth --custom_private=ddcc8e6a9be77249cb44a7d3b... --balance --rpc=ETH_L2_RPC
  ./generateKeys sol --custom_mnemonic="abandon abandon abandon..." --send --amount=0.1 --to=8TinVypdVXQcLoTkr2ezbVumquEoWpt...
```

## Examples

```bash
# Generate Bitcoin key pair
./generateKeys btc

# Generate Bitcoin key pair with 'hoho' string anywhere in public key
./generateKeys btc -i=hoho

# Generate Ethereum key pair with 'xyz' postfix in public key
./generateKeys eth --i=xyz --postfix

# Generate Solana key pair with both prefix and postfix
./generateKeys sol -i=ok --prefix --postfix

# Generate Ethereum key with custom mnemonic
./generateKeys eth --custom_mnemonic="abandon abandon abandon..."

# Check Bitcoin balance
./generateKeys btc --custom_private=5KJvsngHeMpm884wtkJHQtFviFj7BvC4S2S7S8ZvwthXyqqiN --balance

# Send ETH
./generateKeys eth --custom_mnemonic="abandon abandon abandon..." --send --amount=0.1 --to=0x742d35Cc6634C0532925a3b8D4C9db96C4b4d8b6

# Send ERC20 token
./generateKeys eth --custom_private=0x1234567890abcdef... --send --amount=100 --to=0x742d35Cc6634C0532925a3b8D4C9db96C4b4d8b6 --token=0xA0b86a33E6441b6B4b1C2C2C2C2C2C2C2C2C2C2C

# Use custom RPC
./generateKeys eth --custom_mnemonic="abandon abandon abandon..." --balance --rpc=https://mainnet.infura.io/v3/YOUR_PROJECT_ID
```

## License

MIT License - see [LICENSE](LICENSE) file for details.
