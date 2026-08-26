//go:build unit

package tlsfingerprint

import (
	"context"
	"net"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTLSFingerprintDefaultDialerHasBoundedTimeouts(t *testing.T) {
	dialer := newDefaultDialer()
	require.Equal(t, defaultDialTimeout, dialer.Timeout)
	require.Equal(t, defaultDialKeepAlive, dialer.KeepAlive)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	started := time.Now()
	conn, err := NewDialer(nil, nil).baseDialer(ctx, "tcp", "127.0.0.1:1")
	if conn != nil {
		_ = conn.Close()
	}
	require.Error(t, err)
	require.Less(t, time.Since(started), time.Second)
}

func TestTLSFingerprintHTTPProxyBoundsCONNECTResponse(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	accepted := make(chan net.Conn, 1)
	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr == nil {
			accepted <- conn
		}
	}()

	proxyURL, err := url.Parse("http://" + listener.Addr().String())
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	started := time.Now()
	conn, err := NewHTTPProxyDialer(nil, proxyURL).DialTLSContext(ctx, "tcp", "example.com:443")
	if conn != nil {
		_ = conn.Close()
	}
	require.Error(t, err)
	require.Less(t, time.Since(started), time.Second)
	select {
	case conn := <-accepted:
		_ = conn.Close()
	default:
	}
}

func TestTLSFingerprintHandshakeHasBoundedContextDeadline(t *testing.T) {
	client, server := net.Pipe()
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	started := time.Now()
	conn, err := performTLSHandshake(ctx, client, nil, "example.com:443")
	if conn != nil {
		_ = conn.Close()
	}
	require.Error(t, err)
	require.Less(t, time.Since(started), time.Second)
}
