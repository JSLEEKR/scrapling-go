# scrapling-go

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?style=for-the-badge&logo=go&logoColor=white)](https://go.dev)
[![License](https://img.shields.io/badge/License-MIT-green?style=for-the-badge)](LICENSE)
[![Tests](https://img.shields.io/badge/Tests-140-success?style=for-the-badge)](https://github.com/JSLEEKR/scrapling-go)

Adaptive web scraping framework for Go with smart element tracking that survives website redesigns.

## Why This Exists

Web scraping is fragile. A single CSS class rename breaks your scraper. **scrapling-go** solves this by fingerprinting elements and using multi-factor similarity scoring to relocate them even after major DOM restructuring — no code changes needed.

Inspired by [D4Vinci/Scrapling](https://github.com/D4Vinci/Scrapling) (33.5K stars, Python), reimplemented in Go with zero CGo dependencies.

## Key Features

- **Adaptive Element Tracking**: Multi-factor similarity scoring (tag, text, attributes, path, siblings, parent) to relocate elements after website changes
- **CSS & XPath Selectors**: Full CSS selector support via cascadia + custom XPath engine
- **Pseudo-Element Support**: `::text` and `::attr(name)` pseudo-elements matching Scrapy/Parsel syntax
- **HTTP Fetcher**: Browser-like header rotation, cookie handling, proxy support, retry logic
- **SQLite Fingerprint Storage**: WAL-mode SQLite for storing element fingerprints (pure Go, no CGo)
- **SequenceMatcher**: Full port of Python's difflib.SequenceMatcher for string similarity
- **CLI Tool**: Fetch, parse, and track elements from the command line

## Installation

```bash
go install github.com/JSLEEKR/scrapling-go/cmd/scrapling@latest
```

Or as a library:

```bash
go get github.com/JSLEEKR/scrapling-go
```

## Quick Start

### As a Library

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/JSLEEKR/scrapling-go/pkg/fetcher"
    "github.com/JSLEEKR/scrapling-go/pkg/selector"
    "github.com/JSLEEKR/scrapling-go/pkg/storage"
    "github.com/JSLEEKR/scrapling-go/pkg/tracker"
)

func main() {
    // Fetch a page
    f, _ := fetcher.New()
    defer f.Close()

    resp, err := f.Get(context.Background(), "https://example.com")
    if err != nil {
        log.Fatal(err)
    }

    // Parse HTML
    root, _ := resp.Parse()

    // CSS selectors
    headings, _ := selector.CSS(root, "h1")
    for _, h := range headings {
        fmt.Println(h.Text())
    }

    // Extract attributes
    hrefs, _ := selector.CSSAttr(root, "a", "href")
    for _, href := range hrefs {
        fmt.Println(href)
    }

    // Adaptive tracking
    store, _ := storage.New("elements.db")
    defer store.Close()

    tr := tracker.New(store)
    found, _ := tr.Find(root, "https://example.com", "h1")
    if found != nil {
        fmt.Printf("Found: %s\n", found.Text())
    }
}
```

### CLI Usage

```bash
# Fetch a URL and print the HTML
scrapling fetch https://example.com

# Extract elements with CSS selector
scrapling fetch -css "h1" https://example.com

# Extract text only
scrapling fetch -css "p" -text https://example.com

# Extract attributes
scrapling fetch -css "a" -attr "href" https://example.com

# XPath selection
scrapling fetch -xpath "//div[@class='content']" https://example.com

# Parse HTML from stdin
echo "<html><body><h1>Hello</h1></body></html>" | scrapling parse -css "h1" -text

# Track elements across changes
scrapling track -css "div#main" https://example.com
```

## Architecture

```
scrapling-go/
├── cmd/scrapling/        CLI entry point
├── pkg/
│   ├── similarity/       SequenceMatcher (difflib port)
│   ├── parser/           HTML parser (wraps golang.org/x/net/html)
│   ├── selector/         CSS + XPath selector engines
│   ├── storage/          SQLite fingerprint storage
│   ├── tracker/          Adaptive element tracking & scoring
│   └── fetcher/          HTTP client with header rotation
└── internal/
    └── headers/          Browser header generation
```

## Core Concepts

### Adaptive Element Tracking

The core innovation is multi-factor similarity scoring for element relocation:

1. **Fingerprinting**: Each tracked element is serialized into a dict capturing tag, text, attributes, DOM path, parent info, siblings, and children
2. **Scoring**: When relocating, every element on the page is scored against the stored fingerprint across 13 dimensions
3. **Relocation**: The highest-scoring element above a configurable threshold is returned

| Factor | Method |
|--------|--------|
| Tag match | Binary (exact) |
| Text similarity | SequenceMatcher ratio |
| Attribute keys | Set intersection/union |
| Attribute values | Set intersection/union |
| `class` attribute | SequenceMatcher ratio |
| `id` attribute | SequenceMatcher ratio |
| `href` attribute | SequenceMatcher ratio |
| `src` attribute | SequenceMatcher ratio |
| DOM path | SequenceMatcher on path string |
| Parent tag | Binary (exact) |
| Parent attributes | Set intersection/union |
| Parent text | SequenceMatcher ratio |
| Sibling count | Ratio comparison |

### Parser (Adaptable)

The `Adaptable` struct wraps `*html.Node` with rich navigation:

```go
root, _ := parser.Parse(htmlString)
body := parser.Body(root)

// Navigation
elem := body.Find("div")
parent := elem.Parent()
children := elem.Children()
siblings := elem.Siblings()
next := elem.NextSibling()
prev := elem.PrevSibling()
ancestors := elem.Ancestors()

// Content
tag := elem.Tag()           // "div"
text := elem.Text()          // direct text
allText := elem.AllText()    // recursive text
attrs := elem.Attrs()        // map[string]string
html := elem.HTML()          // outer HTML
inner := elem.InnerHTML()    // inner HTML

// Search
divs := elem.FindAll("div")
byAttr := elem.FindByAttr("class", "main")
byText := elem.FindByText("hello")
all := elem.AllElements()

// Metadata
path := elem.Path()          // ["html", "body", "div"]
pathStr := elem.PathString() // "/html/body/div"
depth := elem.Depth()        // 2
```

### CSS Selectors

Full CSS selector support including pseudo-elements:

```go
// Standard CSS selectors
results, _ := selector.CSS(root, "div.container > p")
results, _ := selector.CSS(root, "#main a[href]")
results, _ := selector.CSS(root, "li:nth-child(2)")

// Pseudo-elements (Scrapy/Parsel compatible)
results, _ := selector.CSS(root, "h1::text")
results, _ := selector.CSS(root, "a::attr(href)")

// Convenience functions
first, _ := selector.CSSFirst(root, "h1")
texts, _ := selector.CSSText(root, "p")
hrefs, _ := selector.CSSAttr(root, "a", "href")
```

### XPath Selectors

Common XPath expressions are supported:

```go
// Descendant search
results, _ := selector.XPath(root, "//div")
results, _ := selector.XPath(root, "//a[@href]")
results, _ := selector.XPath(root, `//div[@class='main']`)

// All elements
results, _ := selector.XPath(root, ".//*")

// Direct children
results, _ := selector.XPath(root, "./div")

// First match
first, _ := selector.XPathFirst(root, "//h1")
```

### HTTP Fetcher

Browser-like HTTP client with safety features:

```go
f, _ := fetcher.New(
    fetcher.WithTimeout(10 * time.Second),
    fetcher.WithMaxRetries(3),
    fetcher.WithProxy("http://proxy:8080"),
    fetcher.WithHeaders(customHeaders),
)
defer f.Close()

// GET request
resp, _ := f.Get(ctx, "https://example.com")

// POST request
resp, _ := f.Post(ctx, "https://api.example.com/data", body)

// Response inspection
fmt.Println(resp.StatusCode)     // 200
fmt.Println(resp.OK())           // true
fmt.Println(resp.ContentType())  // "text/html"
fmt.Println(resp.Text())         // body as string

// Parse response HTML
root, _ := resp.Parse()

// Cookie management
f.SetCookies("https://example.com", cookies)
sessionCookie := resp.Cookie("session")
```

### SQLite Storage

Thread-safe fingerprint persistence:

```go
store, _ := storage.New("elements.db")
defer store.Close()

// Save
store.Save("https://example.com", "div#main", &storage.ElementDict{
    Tag:        "div",
    Text:       "Hello",
    Attributes: map[string]string{"id": "main"},
    Path:       []string{"html", "body"},
})

// Load
elem, _ := store.Load("https://example.com", "div#main")

// List all identifiers for a URL
ids, _ := store.List("https://example.com")

// Maintenance
count, _ := store.Count()
store.Clear()
store.Delete("https://example.com", "div#main")
```

### SequenceMatcher

Full port of Python's difflib.SequenceMatcher:

```go
sm := similarity.NewSequenceMatcher("hello world", "hello earth")
ratio := sm.Ratio()                    // 0.727...
blocks := sm.GetMatchingBlocks()       // matching subsequences
opcodes := sm.GetOpcodes()             // edit operations
quick := sm.QuickRatio()               // upper bound estimate

// Convenience
ratio := similarity.StringRatio("abc", "abd")  // 0.667

// Set similarity (Jaccard)
ratio := similarity.SetRatio(
    []string{"a", "b", "c"},
    []string{"b", "c", "d"},
)  // 0.5
```

## Dependencies

| Package | Purpose |
|---------|---------|
| `golang.org/x/net/html` | HTML parsing |
| `github.com/andybalholm/cascadia` | CSS selectors |
| `modernc.org/sqlite` | Pure Go SQLite (no CGo) |

## Comparison with Original

| Feature | Scrapling (Python) | scrapling-go |
|---------|-------------------|--------------|
| Language | Python 3.8+ | Go 1.22+ |
| HTML Parser | lxml | golang.org/x/net/html |
| CSS Selectors | cssselect | cascadia |
| XPath | lxml | Custom engine |
| Element Tracking | Yes (13 factors) | Yes (13 factors) |
| HTTP Client | curl_cffi | net/http |
| Browser Automation | Playwright | Not included |
| Spider Framework | Yes | Not included |
| Storage | SQLite | SQLite (pure Go) |
| Dependencies | 12+ | 3 |
| Binary Size | N/A | ~15MB (single binary) |
| Tests | 92% coverage | 140 tests |

### What We Improved

- **Zero CGo**: Pure Go SQLite via modernc.org/sqlite — no C compiler needed
- **Single Binary**: Compile once, run anywhere — no Python environment
- **Fewer Dependencies**: 3 external deps vs 12+
- **Concurrent Safe**: Proper mutex-based thread safety on storage
- **Type Safe**: Go's type system catches errors at compile time

### What We Skipped

- Browser automation (Playwright integration) — would require chromedp/rod
- Spider framework (concurrent crawling, checkpoints) — would be a separate project
- MCP server integration
- Interactive shell

## Testing

```bash
# Run all tests
go test ./... -v

# Run with race detector
go test ./... -race

# Run specific package
go test ./pkg/tracker/ -v

# Run with coverage
go test ./... -cover
```

## License

MIT License - see [LICENSE](LICENSE)

## Credits

- Original [Scrapling](https://github.com/D4Vinci/Scrapling) by D4Vinci
- Part of the [JSLEEKR V2 Pipeline](https://github.com/JSLEEKR): Reimplement & Compare
