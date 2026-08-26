package proxyutil

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSOCKS5ForwardDialerHasBoundedTimeout(t *testing.T) {
	require.Equal(t, 10*time.Second, socks5ForwardDialer.Timeout)
	require.Equal(t, 30*time.Second, socks5ForwardDialer.KeepAlive)
}

func TestConfigureTransportProxySOCKS5SetsDialContext(t *testing.T) {
	for _, scheme := range []string{"socks5", "socks5h"} {
		t.Run(scheme, func(t *testing.T) {
			u, err := url.Parse(scheme + "://127.0.0.1:1080")
			require.NoError(t, err)
			transport := &http.Transport{}
			require.NoError(t, ConfigureTransportProxy(transport, u))
			require.NotNil(t, transport.DialContext)
			require.Nil(t, transport.Proxy)
		})
	}
}

func TestConfigureTransportProxyHTTPPreservesDialContext(t *testing.T) {
	u, err := url.Parse("http://127.0.0.1:8080")
	require.NoError(t, err)
	called := false
	transport := &http.Transport{DialContext: func(context.Context, string, string) (net.Conn, error) {
		called = true
		return nil, context.Canceled
	}}
	require.NoError(t, ConfigureTransportProxy(transport, u))
	_, _ = transport.DialContext(context.Background(), "tcp", "127.0.0.1:1")
	require.True(t, called)
}
