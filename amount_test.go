package main

import "testing"

func TestParseAmountToBaseUnits(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		amount   string
		decimals int
		want     string
	}{
		{name: "whole", amount: "2", decimals: 8, want: "200000000"},
		{name: "fraction", amount: "0.00000001", decimals: 8, want: "1"},
		{name: "trims spaces", amount: " 1.5 ", decimals: 9, want: "1500000000"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseAmountToBaseUnits(tt.amount, tt.decimals)
			if err != nil {
				t.Fatalf("parseAmountToBaseUnits() error = %v", err)
			}
			if got.String() != tt.want {
				t.Fatalf("parseAmountToBaseUnits() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestParseAmountToBaseUnitsRejectsTooMuchPrecision(t *testing.T) {
	t.Parallel()

	if _, err := parseAmountToBaseUnits("0.000000001", 8); err == nil {
		t.Fatal("expected precision error")
	}
}
