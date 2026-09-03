package skillimport

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type scriptedFetcher struct {
	mu      sync.Mutex
	calls   []string
	handler func(string, FetchOptions) (FetchResult, error)
}

func (f *scriptedFetcher) Get(_ context.Context, target string, options FetchOptions) (FetchResult, error) {
	f.mu.Lock()
	f.calls = append(f.calls, target)
	f.mu.Unlock()
	if f.handler == nil {
		return FetchResult{}, fmt.Errorf("unexpected fetch %s", target)
	}
	return f.handler(target, options)
}

func (f *scriptedFetcher) callCount(fragment string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	count := 0
	for _, call := range f.calls {
		if strings.Contains(call, fragment) {
			count++
		}
	}
	return count
}

func fetchResult(target, contentType string, body []byte) FetchResult {
	return FetchResult{Body: body, Evidence: Evidence{
		URL: target, CapturedAt: time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC),
		SHA256: digestBytes(body), ByteSize: int64(len(body)), ContentType: contentType,
	}}
}

func makeSourceZIP(entries map[string][]byte) []byte {
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for name, data := range entries {
		header := &zip.FileHeader{Name: name, Method: zip.Deflate}
		header.SetMode(0o644)
		entry, _ := writer.CreateHeader(header)
		_, _ = entry.Write(data)
	}
	_ = writer.Close()
	return buffer.Bytes()
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func testSecureFetcher(client *http.Client) *SecureFetcher {
	return &SecureFetcher{
		client: client, timeout: time.Second, maxResponseBytes: 1024 * 1024,
		userAgent: "test", semaphore: make(chan struct{}, 4), maxPerHost: 2,
		hosts: make(map[string]*hostGate), validateTarget: validateHTTPSImportTarget,
		now: func() time.Time { return time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC) },
	}
}

func response(request *http.Request, status int, headers http.Header, body string) *http.Response {
	if headers == nil {
		headers = make(http.Header)
	}
	return &http.Response{
		StatusCode: status, Header: headers, Body: ioNopCloser{bytes.NewBufferString(body)},
		Request: request, ContentLength: int64(len(body)),
	}
}

type ioNopCloser struct{ *bytes.Buffer }

func (ioNopCloser) Close() error { return nil }

func mustURL(raw string) *url.URL {
	parsed, err := url.Parse(raw)
	if err != nil {
		panic(err)
	}
	return parsed
}
