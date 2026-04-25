package main

import (
	"fmt"
	"math/big"
	"strings"
)

func parseAmountToBaseUnits(amount string, decimals int) (*big.Int, error) {
	amount = strings.TrimSpace(amount)
	if amount == "" {
		return nil, fmt.Errorf("amount is required")
	}
	if decimals < 0 {
		return nil, fmt.Errorf("invalid decimals: %d", decimals)
	}

	value, ok := new(big.Rat).SetString(amount)
	if !ok || value.Sign() <= 0 {
		return nil, fmt.Errorf("amount must be a positive decimal number")
	}

	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	scaled := new(big.Rat).Mul(value, new(big.Rat).SetInt(scale))
	if !scaled.IsInt() {
		return nil, fmt.Errorf("amount has more than %d decimal places", decimals)
	}

	return new(big.Int).Set(scaled.Num()), nil
}

func parseAmountToInt64(amount string, decimals int) (int64, error) {
	baseUnits, err := parseAmountToBaseUnits(amount, decimals)
	if err != nil {
		return 0, err
	}
	if !baseUnits.IsInt64() {
		return 0, fmt.Errorf("amount is too large")
	}
	return baseUnits.Int64(), nil
}

func parseAmountToUint64(amount string, decimals int) (uint64, error) {
	baseUnits, err := parseAmountToBaseUnits(amount, decimals)
	if err != nil {
		return 0, err
	}
	if !baseUnits.IsUint64() {
		return 0, fmt.Errorf("amount is too large")
	}
	return baseUnits.Uint64(), nil
}
