// Package headers provides browser header generation for HTTP requests.
// It generates realistic browser headers with proper Accept, User-Agent,
// and other fields to mimic real browser traffic patterns.
package headers

import (
	"math/rand"
	"runtime"
)

// BrowserType represents a browser family.
type BrowserType string

const (
	Chrome  BrowserType = "chrome"
	Firefox BrowserType = "firefox"
	Edge    BrowserType = "edge"
)

// Header represents an HTTP header key-value pair.
type Header struct {
	Key   string
	Value string
}

// Generate creates a set of browser-like HTTP headers.
func Generate(browser BrowserType) []Header {
	headers := []Header{
		{Key: "User-Agent", Value: userAgent(browser)},
		{Key: "Accept", Value: "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8"},
		{Key: "Accept-Language", Value: acceptLanguage()},
		{Key: "Accept-Encoding", Value: "gzip, deflate, br"},
		{Key: "Connection", Value: "keep-alive"},
		{Key: "Upgrade-Insecure-Requests", Value: "1"},
		{Key: "Sec-Fetch-Dest", Value: "document"},
		{Key: "Sec-Fetch-Mode", Value: "navigate"},
		{Key: "Sec-Fetch-Site", Value: "none"},
		{Key: "Sec-Fetch-User", Value: "?1"},
		{Key: "Cache-Control", Value: "max-age=0"},
	}
	return headers
}

// GenerateRandom creates headers with a random browser type.
func GenerateRandom() []Header {
	browsers := []BrowserType{Chrome, Firefox, Edge}
	return Generate(browsers[rand.Intn(len(browsers))])
}

func userAgent(browser BrowserType) string {
	os := osString()
	switch browser {
	case Firefox:
		versions := []string{"126.0", "125.0", "124.0"}
		ver := versions[rand.Intn(len(versions))]
		return "Mozilla/5.0 (" + os + "; rv:" + ver + ") Gecko/20100101 Firefox/" + ver
	case Edge:
		versions := []string{"124.0.0.0", "123.0.0.0", "122.0.0.0"}
		ver := versions[rand.Intn(len(versions))]
		return "Mozilla/5.0 (" + os + ") AppleWebKit/537.36 (KHTML, like Gecko) Chrome/" + ver + " Safari/537.36 Edg/" + ver
	default: // Chrome
		versions := []string{"125.0.0.0", "124.0.0.0", "123.0.0.0"}
		ver := versions[rand.Intn(len(versions))]
		return "Mozilla/5.0 (" + os + ") AppleWebKit/537.36 (KHTML, like Gecko) Chrome/" + ver + " Safari/537.36"
	}
}

func osString() string {
	switch runtime.GOOS {
	case "windows":
		return "Windows NT 10.0; Win64; x64"
	case "darwin":
		return "Macintosh; Intel Mac OS X 10_15_7"
	default:
		return "X11; Linux x86_64"
	}
}

func acceptLanguage() string {
	langs := []string{
		"en-US,en;q=0.9",
		"en-US,en;q=0.9,ko;q=0.8",
		"en-GB,en;q=0.9",
		"en-US,en;q=0.5",
	}
	return langs[rand.Intn(len(langs))]
}
