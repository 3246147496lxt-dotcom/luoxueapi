package repository

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseProxyIDSearch(t *testing.T) {
	tests := []struct {
		input string
		id    int64
		ok    bool
	}{
		{input: "#42", id: 42, ok: true},
		{input: "  #7  ", id: 7, ok: true},
		{input: "42", ok: false},
		{input: "#0", ok: false},
		{input: "#-1", ok: false},
		{input: "#abc", ok: false},
		{input: "edge #42", ok: false},
	}
	for _, test := range tests {
		id, ok := parseProxyIDSearch(test.input)
		require.Equal(t, test.ok, ok, test.input)
		require.Equal(t, test.id, id, test.input)
	}
}
