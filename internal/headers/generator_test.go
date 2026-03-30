package headers

import (
	"strings"
	"testing"
)

func TestGenerateChrome(t *testing.T) {
	h := Generate(Chrome)
	if len(h) == 0 {
		t.Fatal("expected headers")
	}

	var ua string
	for _, hdr := range h {
		if hdr.Key == "User-Agent" {
			ua = hdr.Value
			break
		}
	}
	if !strings.Contains(ua, "Chrome") {
		t.Errorf("expected Chrome in UA, got %s", ua)
	}
}

func TestGenerateFirefox(t *testing.T) {
	h := Generate(Firefox)
	var ua string
	for _, hdr := range h {
		if hdr.Key == "User-Agent" {
			ua = hdr.Value
			break
		}
	}
	if !strings.Contains(ua, "Firefox") {
		t.Errorf("expected Firefox in UA, got %s", ua)
	}
}

func TestGenerateEdge(t *testing.T) {
	h := Generate(Edge)
	var ua string
	for _, hdr := range h {
		if hdr.Key == "User-Agent" {
			ua = hdr.Value
			break
		}
	}
	if !strings.Contains(ua, "Edg") {
		t.Errorf("expected Edg in UA, got %s", ua)
	}
}

func TestGenerateRandom(t *testing.T) {
	h := GenerateRandom()
	if len(h) == 0 {
		t.Fatal("expected headers")
	}
	// Should have at least Accept, User-Agent
	hasUA, hasAccept := false, false
	for _, hdr := range h {
		if hdr.Key == "User-Agent" {
			hasUA = true
		}
		if hdr.Key == "Accept" {
			hasAccept = true
		}
	}
	if !hasUA {
		t.Error("expected User-Agent header")
	}
	if !hasAccept {
		t.Error("expected Accept header")
	}
}

func TestGenerateHasSecurityHeaders(t *testing.T) {
	h := Generate(Chrome)
	secHeaders := map[string]bool{
		"Sec-Fetch-Dest": false,
		"Sec-Fetch-Mode": false,
		"Sec-Fetch-Site": false,
	}
	for _, hdr := range h {
		if _, ok := secHeaders[hdr.Key]; ok {
			secHeaders[hdr.Key] = true
		}
	}
	for key, found := range secHeaders {
		if !found {
			t.Errorf("missing security header: %s", key)
		}
	}
}
