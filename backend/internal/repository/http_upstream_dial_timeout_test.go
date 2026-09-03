package repository

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBuildUpstreamTransportSetsDialTimeout(t *testing.T) {
	transport, err := buildUpstreamTransport(defaultPoolSettings(nil), nil, upstreamProtocolModeDefault)
	require.NoError(t, err)
	require.NotNil(t, transport.DialContext)
	require.Equal(t, defaultUpstreamTLSHandshakeTimeout, transport.TLSHandshakeTimeout)
}

func TestNewUpstreamDialerHasBoundedTimeout(t *testing.T) {
	dialer := newUpstreamDialer()
	require.Equal(t, 10*time.Second, dialer.Timeout)
	require.Equal(t, 30*time.Second, dialer.KeepAlive)
}

func TestBuildUpstreamTransportKeepsDialTimeoutWithProxy(t *testing.T) {
	u, err := url.Parse("http://127.0.0.1:1080")
	require.NoError(t, err)
	transport, err := buildUpstreamTransport(defaultPoolSettings(nil), u, upstreamProtocolModeDefault)
	require.NoError(t, err)
	require.NotNil(t, transport.DialContext)
}

func TestUpstreamDialerRespectsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	conn, err := newUpstreamDialer().DialContext(ctx, "tcp", "127.0.0.1:1")
	if conn != nil {
		_ = conn.Close()
	}
	require.Error(t, err)
}
