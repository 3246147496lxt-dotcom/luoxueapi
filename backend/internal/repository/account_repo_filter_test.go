package repository

import "testing"

func TestParseAccountListSpecialSearch(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantAccount int64
		wantProxy   int64
	}{
		{name: "account id", input: "#42", wantAccount: 42},
		{name: "account id trims whitespace", input: "  # 7  ", wantAccount: 7},
		{name: "proxy id", input: "proxy:19", wantProxy: 19},
		{name: "proxy prefix is case insensitive", input: "PROXY: 23", wantProxy: 23},
		{name: "normal search", input: "customer #42"},
		{name: "invalid account id falls back", input: "#0"},
		{name: "invalid proxy id falls back", input: "proxy:not-a-number"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			accountID, proxyID := parseAccountListSpecialSearch(tt.input)
			if tt.wantAccount == 0 {
				if accountID != nil {
					t.Fatalf("account id = %d, want nil", *accountID)
				}
			} else if accountID == nil || *accountID != tt.wantAccount {
				t.Fatalf("account id = %v, want %d", accountID, tt.wantAccount)
			}
			if tt.wantProxy == 0 {
				if proxyID != nil {
					t.Fatalf("proxy id = %d, want nil", *proxyID)
				}
			} else if proxyID == nil || *proxyID != tt.wantProxy {
				t.Fatalf("proxy id = %v, want %d", proxyID, tt.wantProxy)
			}
		})
	}
}
