# miniw - mini wallet

**miniw** is a simple but powerful lightweight crypto mini wallet and key generation application that generates self-custodial wallet key pairs and manages cryptocurrency transactions for the following blockchains:

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
$ ./miniw
Usage: ./miniw <NETWORK> | [WALLET] | <ACTION>

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
  --privatekey             Use custom private key
  --mnemonic               Use custom mnemonic
  --path                   Use custom derivation path (Optional)

Action Commands
  --balance <wallet>       Check balance for wallet address
  --send                   Send cryptocurrency (requires --privatekey or --mnemonic)
    --amount <amount>      Amount to send
    --to <address>         Destination address
      --token <address>    Custom Token contract address (Optional)
  --rpc <url>              Custom RPC endpoint for the chosen network (Optional)

Examples:
  ./miniw btc --balance=1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa
  ./miniw eth --balance=0x742d35Cc6634C0532925a3b8D4C9db96C4b4d8b6 --rpc=ETH_L2_RPC
  ./miniw sol --mnemonic="abandon abandon abandon..." --send --amount=0.1 --to=8TinVypdVXQcLoTkr2ezbVumquEoWpt...
```

## Examples

```bash
# Generate Bitcoin key pair
./miniw btc

# Generate Bitcoin key pair with 'hoho' string anywhere in public key
./miniw btc -i=hoho

# Generate Ethereum key pair with 'xyz' postfix in public key
./miniw eth -i=xyz --postfix

# Generate Solana key pair with both prefix and postfix
./miniw sol -i=ok --prefix --postfix

# Generate Ethereum key with custom mnemonic
./miniw eth --mnemonic="abandon abandon abandon..."

# Check Bitcoin balance
./miniw btc --balance=1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa

# Send ETH
./miniw eth --mnemonic="abandon abandon abandon..." --send --amount=0.1 --to=0x742d35Cc6634C0532925a3b8D4C9db96C4b4d8b6

# Send ERC20 token
./miniw eth --privatekey=0x1234567890abcdef... --send --amount=100 --to=0x742d35Cc6634C0532925a3b8D4C9db96C4b4d8b6 --token=0xA0b86a33E6441b6B4b1C2C2C2C2C2C2C2C2C2C2C

# Use custom RPC
./miniw eth --balance=0x742d35Cc6634C0532925a3b8D4C9db96C4b4d8b6 --rpc=https://mainnet.infura.io/v3/YOUR_PROJECT_ID
```

## License

MIT License - see [LICENSE](LICENSE) file for details.
