// Command scrapling is a CLI tool for web scraping with adaptive element tracking.
// It can fetch URLs, parse HTML, select elements via CSS/XPath, and track
// elements across website changes using similarity-based fingerprinting.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/JSLEEKR/scrapling-go/pkg/fetcher"
	"github.com/JSLEEKR/scrapling-go/pkg/parser"
	"github.com/JSLEEKR/scrapling-go/pkg/selector"
	"github.com/JSLEEKR/scrapling-go/pkg/storage"
	"github.com/JSLEEKR/scrapling-go/pkg/tracker"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		printUsage()
		return nil
	}

	switch args[0] {
	case "fetch":
		return cmdFetch(args[1:])
	case "parse":
		return cmdParse(args[1:])
	case "track":
		return cmdTrack(args[1:])
	case "version":
		fmt.Println("scrapling-go v1.0.0")
		return nil
	case "help", "-h", "--help":
		printUsage()
		return nil
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func cmdFetch(args []string) error {
	fs := flag.NewFlagSet("fetch", flag.ContinueOnError)
	cssFlag := fs.String("css", "", "CSS selector to extract")
	xpathFlag := fs.String("xpath", "", "XPath expression to extract")
	textOnly := fs.Bool("text", false, "Extract text content only")
	attrFlag := fs.String("attr", "", "Extract attribute value")
	timeout := fs.Duration("timeout", 30*time.Second, "Request timeout")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: scrapling fetch [options] <url>")
	}

	url := fs.Arg(0)
	f, err := fetcher.New(fetcher.WithTimeout(*timeout))
	if err != nil {
		return fmt.Errorf("create fetcher: %w", err)
	}
	defer f.Close()

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	resp, err := f.Get(ctx, url)
	if err != nil {
		return fmt.Errorf("fetch %s: %w", url, err)
	}

	fmt.Fprintf(os.Stderr, "Status: %d\n", resp.StatusCode)
	fmt.Fprintf(os.Stderr, "Content-Type: %s\n", resp.ContentType())
	fmt.Fprintf(os.Stderr, "Body length: %d bytes\n\n", len(resp.Body))

	if *cssFlag == "" && *xpathFlag == "" {
		fmt.Println(resp.Text())
		return nil
	}

	root, err := resp.Parse()
	if err != nil {
		return fmt.Errorf("parse html: %w", err)
	}

	return extractAndPrint(root, *cssFlag, *xpathFlag, *textOnly, *attrFlag)
}

func cmdParse(args []string) error {
	fs := flag.NewFlagSet("parse", flag.ContinueOnError)
	cssFlag := fs.String("css", "", "CSS selector")
	xpathFlag := fs.String("xpath", "", "XPath expression")
	textOnly := fs.Bool("text", false, "Text content only")
	attrFlag := fs.String("attr", "", "Extract attribute")

	if err := fs.Parse(args); err != nil {
		return err
	}

	// Read HTML from stdin
	var sb strings.Builder
	buf := make([]byte, 4096)
	for {
		n, err := os.Stdin.Read(buf)
		if n > 0 {
			sb.Write(buf[:n])
		}
		if err != nil {
			break
		}
	}

	root, err := parser.Parse(sb.String())
	if err != nil {
		return fmt.Errorf("parse html: %w", err)
	}

	if *cssFlag == "" && *xpathFlag == "" {
		body := parser.Body(root)
		if body != nil {
			fmt.Println(body.AllText())
		}
		return nil
	}

	return extractAndPrint(root, *cssFlag, *xpathFlag, *textOnly, *attrFlag)
}

func cmdTrack(args []string) error {
	fs := flag.NewFlagSet("track", flag.ContinueOnError)
	dbPath := fs.String("db", "elements.db", "SQLite database path")
	cssFlag := fs.String("css", "", "CSS selector to track")
	threshold := fs.Float64("threshold", 50.0, "Similarity threshold (0-100)")
	timeout := fs.Duration("timeout", 30*time.Second, "Request timeout")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() < 1 || *cssFlag == "" {
		return fmt.Errorf("usage: scrapling track -css <selector> <url>")
	}

	url := fs.Arg(0)
	store, err := storage.New(*dbPath)
	if err != nil {
		return fmt.Errorf("open storage: %w", err)
	}
	defer store.Close()

	tr := tracker.New(store)
	tr.SetThreshold(*threshold)

	f, err := fetcher.New(fetcher.WithTimeout(*timeout))
	if err != nil {
		return fmt.Errorf("create fetcher: %w", err)
	}
	defer f.Close()

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	resp, err := f.Get(ctx, url)
	if err != nil {
		return fmt.Errorf("fetch: %w", err)
	}

	root, err := resp.Parse()
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	found, err := tr.Find(root, url, *cssFlag)
	if err != nil {
		return fmt.Errorf("find element: %w", err)
	}
	if found == nil {
		fmt.Println("Element not found")
		return nil
	}

	fmt.Printf("Found: <%s>\n", found.Tag())
	fmt.Printf("Text: %s\n", found.Text())
	fmt.Printf("Path: %s\n", found.PathString())
	fmt.Printf("Attrs: %v\n", found.Attrs())

	// Show top matches for relocation context
	matches, err := tr.TopMatches(root, url, *cssFlag, 5)
	if err == nil && len(matches) > 0 {
		fmt.Println("\nTop matches by similarity:")
		for i, m := range matches {
			fmt.Printf("  %d. <%s> score=%.2f text=%q\n", i+1, m.Element.Tag(), m.Score, truncate(m.Element.Text(), 50))
		}
	}

	return nil
}

func extractAndPrint(root *parser.Adaptable, css, xpath string, textOnly bool, attr string) error {
	var results []*parser.Adaptable
	var err error

	if css != "" {
		results, err = selector.CSS(root, css)
	} else if xpath != "" {
		results, err = selector.XPath(root, xpath)
	}
	if err != nil {
		return err
	}

	for _, r := range results {
		switch {
		case attr != "":
			fmt.Println(r.Attr(attr))
		case textOnly:
			fmt.Println(r.AllText())
		default:
			fmt.Println(r.HTML())
		}
	}

	fmt.Fprintf(os.Stderr, "\n(%d results)\n", len(results))
	return nil
}

func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "..."
}

func printUsage() {
	fmt.Println(`scrapling-go - Adaptive web scraping framework

Usage:
  scrapling <command> [options]

Commands:
  fetch   Fetch a URL and optionally extract elements
  parse   Parse HTML from stdin and extract elements
  track   Track elements across website changes
  version Show version
  help    Show this help

Examples:
  scrapling fetch https://example.com
  scrapling fetch -css "h1" https://example.com
  scrapling fetch -css "a" -attr "href" https://example.com
  scrapling fetch -xpath "//div[@class='content']" -text https://example.com
  echo "<html>..." | scrapling parse -css "p"
  scrapling track -css "div#main" https://example.com`)
}
