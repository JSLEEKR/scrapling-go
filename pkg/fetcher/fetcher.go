// Package fetcher provides an HTTP client with header rotation, proxy support,
// cookie handling, and retry logic. It produces unified Response objects that
// integrate with the parser for immediate HTML extraction.
package fetcher

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

// Fetcher is an HTTP client with browser-like header rotation and retry logic.
type Fetcher struct {
	client     *http.Client
	headers    http.Header
	maxRetries int
	timeout    time.Duration
	proxyURL   string
}

// Option configures a Fetcher.
type Option func(*Fetcher)

// WithTimeout sets the HTTP request timeout.
func WithTimeout(d time.Duration) Option {
	return func(f *Fetcher) {
		f.timeout = d
	}
}

// WithMaxRetries sets the maximum number of retries.
func WithMaxRetries(n int) Option {
	return func(f *Fetcher) {
		f.maxRetries = n
	}
}

// WithProxy sets the proxy URL.
func WithProxy(proxyURL string) Option {
	return func(f *Fetcher) {
		f.proxyURL = proxyURL
	}
}

// WithHeaders sets custom HTTP headers.
func WithHeaders(headers http.Header) Option {
	return func(f *Fetcher) {
		for k, v := range headers {
			f.headers[k] = v
		}
	}
}

// New creates a new Fetcher with the given options.
func New(opts ...Option) (*Fetcher, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("create cookie jar: %w", err)
	}

	f := &Fetcher{
		headers:    make(http.Header),
		maxRetries: 3,
		timeout:    30 * time.Second,
	}

	for _, opt := range opts {
		opt(f)
	}

	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	}

	if f.proxyURL != "" {
		proxyParsed, err := url.Parse(f.proxyURL)
		if err != nil {
			return nil, fmt.Errorf("parse proxy url: %w", err)
		}
		transport.Proxy = http.ProxyURL(proxyParsed)
	}

	f.client = &http.Client{
		Jar:       jar,
		Transport: transport,
		Timeout:   f.timeout,
	}

	return f, nil
}

// Get performs an HTTP GET request.
func (f *Fetcher) Get(ctx context.Context, rawURL string) (*Response, error) {
	return f.Do(ctx, http.MethodGet, rawURL, nil)
}

// Post performs an HTTP POST request.
func (f *Fetcher) Post(ctx context.Context, rawURL string, body io.Reader) (*Response, error) {
	return f.Do(ctx, http.MethodPost, rawURL, body)
}

// Do performs an HTTP request with retry logic and header rotation.
// It retries on network errors and HTTP 5xx responses. For POST/PUT/PATCH
// requests with a body, the body is buffered so it can be replayed on retries.
func (f *Fetcher) Do(ctx context.Context, method, rawURL string, body io.Reader) (*Response, error) {
	// Buffer the body so it can be replayed on retries
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = io.ReadAll(body)
		if err != nil {
			return nil, fmt.Errorf("read request body: %w", err)
		}
	}

	var lastErr error
	var lastResp *Response

	for attempt := 0; attempt <= f.maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			backoff := time.Duration(1<<uint(attempt-1)) * 100 * time.Millisecond
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		var bodyReader io.Reader
		if bodyBytes != nil {
			bodyReader = bytes.NewReader(bodyBytes)
		}

		req, err := http.NewRequestWithContext(ctx, method, rawURL, bodyReader)
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}

		// Apply headers
		f.applyHeaders(req)

		resp, err := f.client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}

		// Extract cookies
		u, _ := url.Parse(rawURL)
		cookies := f.client.Jar.Cookies(u)

		r := &Response{
			URL:        resp.Request.URL.String(),
			StatusCode: resp.StatusCode,
			Headers:    resp.Header,
			Body:       respBody,
			Encoding:   detectEncoding(resp.Header),
			Cookies:    cookies,
		}

		// Retry on 5xx server errors
		if resp.StatusCode >= 500 && attempt < f.maxRetries {
			lastErr = fmt.Errorf("server error: %d", resp.StatusCode)
			lastResp = r
			continue
		}

		return r, nil
	}

	// If last attempt got a response (e.g., repeated 5xx), return it
	if lastResp != nil {
		return lastResp, nil
	}

	return nil, fmt.Errorf("request failed after %d retries: %w", f.maxRetries+1, lastErr)
}

// applyHeaders sets browser-like headers on the request.
func (f *Fetcher) applyHeaders(req *http.Request) {
	// Set custom headers first
	for k, v := range f.headers {
		req.Header[k] = v
	}

	// Set default browser-like headers if not already set
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", randomUserAgent())
	}
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	}
	if req.Header.Get("Accept-Language") == "" {
		req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	}
	if req.Header.Get("Accept-Encoding") == "" {
		req.Header.Set("Accept-Encoding", "gzip, deflate")
	}
	if req.Header.Get("Connection") == "" {
		req.Header.Set("Connection", "keep-alive")
	}
}

// Close releases resources held by the fetcher.
func (f *Fetcher) Close() {
	f.client.CloseIdleConnections()
}

// SetCookies sets cookies for a given URL.
func (f *Fetcher) SetCookies(rawURL string, cookies []*http.Cookie) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("parse url: %w", err)
	}
	f.client.Jar.SetCookies(u, cookies)
	return nil
}

// detectEncoding extracts the charset from Content-Type header.
func detectEncoding(headers http.Header) string {
	ct := headers.Get("Content-Type")
	if ct == "" {
		return "utf-8"
	}
	for _, part := range strings.Split(ct, ";") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "charset=") {
			return strings.TrimPrefix(part, "charset=")
		}
	}
	return "utf-8"
}

// Chrome user agents for rotation.
var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:126.0) Gecko/20100101 Firefox/126.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:126.0) Gecko/20100101 Firefox/126.0",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36 Edg/124.0.0.0",
}

func randomUserAgent() string {
	return userAgents[rand.Intn(len(userAgents))]
}
