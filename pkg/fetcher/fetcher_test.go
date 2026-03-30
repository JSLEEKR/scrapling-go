package fetcher

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewFetcher(t *testing.T) {
	f, err := New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f == nil {
		t.Fatal("expected non-nil fetcher")
	}
	f.Close()
}

func TestNewFetcherWithOptions(t *testing.T) {
	f, err := New(
		WithTimeout(10*time.Second),
		WithMaxRetries(5),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.maxRetries != 5 {
		t.Errorf("expected maxRetries=5, got %d", f.maxRetries)
	}
	f.Close()
}

func TestGet(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte("<html><body><h1>Hello</h1></body></html>"))
	}))
	defer ts.Close()

	f, err := New()
	if err != nil {
		t.Fatalf("create fetcher: %v", err)
	}
	defer f.Close()

	resp, err := f.Get(context.Background(), ts.URL)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(resp.Text(), "Hello") {
		t.Error("expected body to contain 'Hello'")
	}
}

func TestGetParseHTML(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("<html><body><div id='main'>Content</div></body></html>"))
	}))
	defer ts.Close()

	f, _ := New()
	defer f.Close()

	resp, err := f.Get(context.Background(), ts.URL)
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	root, err := resp.Parse()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if root == nil {
		t.Fatal("expected parsed root")
	}
}

func TestPost(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.Write([]byte("OK"))
	}))
	defer ts.Close()

	f, _ := New()
	defer f.Close()

	resp, err := f.Post(context.Background(), ts.URL, strings.NewReader("data=test"))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestRetryOnFailure(t *testing.T) {
	attempts := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write([]byte("OK"))
	}))
	defer ts.Close()

	f, _ := New(WithMaxRetries(0)) // No retries
	defer f.Close()

	resp, err := f.Get(context.Background(), ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// With 0 retries and first attempt returning 500, should get 500
	if resp.StatusCode != 500 {
		t.Errorf("expected 500 with no retries, got %d", resp.StatusCode)
	}
}

func TestUserAgentRotation(t *testing.T) {
	var receivedUA string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedUA = r.Header.Get("User-Agent")
		w.Write([]byte("OK"))
	}))
	defer ts.Close()

	f, _ := New()
	defer f.Close()

	_, err := f.Get(context.Background(), ts.URL)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if receivedUA == "" {
		t.Error("expected User-Agent header")
	}
}

func TestCustomHeaders(t *testing.T) {
	var receivedHeader string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeader = r.Header.Get("X-Custom")
		w.Write([]byte("OK"))
	}))
	defer ts.Close()

	h := make(http.Header)
	h.Set("X-Custom", "test-value")
	f, _ := New(WithHeaders(h))
	defer f.Close()

	_, err := f.Get(context.Background(), ts.URL)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if receivedHeader != "test-value" {
		t.Errorf("expected X-Custom=test-value, got %s", receivedHeader)
	}
}

func TestSetCookies(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("test")
		if err != nil {
			w.WriteHeader(400)
			return
		}
		w.Write([]byte(cookie.Value))
	}))
	defer ts.Close()

	f, _ := New()
	defer f.Close()

	err := f.SetCookies(ts.URL, []*http.Cookie{
		{Name: "test", Value: "hello"},
	})
	if err != nil {
		t.Fatalf("set cookies: %v", err)
	}

	resp, err := f.Get(context.Background(), ts.URL)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if resp.Text() != "hello" {
		t.Errorf("expected cookie value 'hello', got '%s'", resp.Text())
	}
}

func TestResponseOK(t *testing.T) {
	resp := &Response{StatusCode: 200}
	if !resp.OK() {
		t.Error("expected OK for 200")
	}

	resp = &Response{StatusCode: 404}
	if resp.OK() {
		t.Error("expected not OK for 404")
	}
}

func TestResponseContentType(t *testing.T) {
	resp := &Response{
		Headers: http.Header{"Content-Type": []string{"text/html"}},
	}
	if resp.ContentType() != "text/html" {
		t.Errorf("expected text/html, got %s", resp.ContentType())
	}
}

func TestResponseContentLength(t *testing.T) {
	resp := &Response{
		Headers: http.Header{"Content-Length": []string{"42"}},
	}
	if resp.ContentLength() != 42 {
		t.Errorf("expected 42, got %d", resp.ContentLength())
	}

	resp2 := &Response{Headers: http.Header{}}
	if resp2.ContentLength() != -1 {
		t.Errorf("expected -1 for missing, got %d", resp2.ContentLength())
	}
}

func TestResponseCookie(t *testing.T) {
	resp := &Response{
		Cookies: []*http.Cookie{
			{Name: "session", Value: "abc123"},
			{Name: "theme", Value: "dark"},
		},
	}
	c := resp.Cookie("session")
	if c == nil || c.Value != "abc123" {
		t.Error("expected session cookie with value abc123")
	}
	if resp.Cookie("nonexistent") != nil {
		t.Error("expected nil for nonexistent cookie")
	}
}

func TestDetectEncoding(t *testing.T) {
	tests := []struct {
		ct       string
		expected string
	}{
		{"text/html; charset=utf-8", "utf-8"},
		{"text/html; charset=iso-8859-1", "iso-8859-1"},
		{"text/html", "utf-8"},
		{"", "utf-8"},
	}
	for _, tt := range tests {
		h := http.Header{}
		if tt.ct != "" {
			h.Set("Content-Type", tt.ct)
		}
		result := detectEncoding(h)
		if result != tt.expected {
			t.Errorf("detectEncoding(%q) = %q, want %q", tt.ct, result, tt.expected)
		}
	}
}

func TestContextCancellation(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		w.Write([]byte("OK"))
	}))
	defer ts.Close()

	f, _ := New(WithTimeout(10 * time.Second))
	defer f.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := f.Get(ctx, ts.URL)
	if err == nil {
		t.Error("expected error from cancelled context")
	}
}

func TestRetryOn5xx(t *testing.T) {
	attempts := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("error"))
			return
		}
		w.Write([]byte("OK"))
	}))
	defer ts.Close()

	f, _ := New(WithMaxRetries(3))
	defer f.Close()

	resp, err := f.Get(context.Background(), ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200 after retry, got %d", resp.StatusCode)
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestRetryPostBodyReplay(t *testing.T) {
	attempts := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		body, _ := io.ReadAll(r.Body)
		if attempts < 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Write(body)
	}))
	defer ts.Close()

	f, _ := New(WithMaxRetries(3))
	defer f.Close()

	resp, err := f.Post(context.Background(), ts.URL, strings.NewReader("test-data"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Text() != "test-data" {
		t.Errorf("expected body 'test-data' after retry, got '%s'", resp.Text())
	}
}

func TestContentLengthOverflow(t *testing.T) {
	resp := &Response{
		Headers: http.Header{"Content-Length": []string{"99999999999999999999"}},
	}
	if resp.ContentLength() != -1 {
		t.Errorf("expected -1 for overflow, got %d", resp.ContentLength())
	}
}
