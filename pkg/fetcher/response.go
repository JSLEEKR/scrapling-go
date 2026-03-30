package fetcher

import (
	"net/http"

	"github.com/JSLEEKR/scrapling-go/pkg/parser"
)

// Response wraps an HTTP response with parsed HTML body access.
type Response struct {
	URL        string
	StatusCode int
	Headers    http.Header
	Body       []byte
	Encoding   string
	Cookies    []*http.Cookie
}

// Text returns the response body as a string.
func (r *Response) Text() string {
	return string(r.Body)
}

// Parse parses the response body as HTML and returns the root Adaptable node.
func (r *Response) Parse() (*parser.Adaptable, error) {
	return parser.Parse(string(r.Body))
}

// OK returns true if the status code is 2xx.
func (r *Response) OK() bool {
	return r.StatusCode >= 200 && r.StatusCode < 300
}

// ContentType returns the Content-Type header value.
func (r *Response) ContentType() string {
	return r.Headers.Get("Content-Type")
}

// ContentLength returns the Content-Length header as int, or -1 if not set
// or if the value overflows.
func (r *Response) ContentLength() int {
	cl := r.Headers.Get("Content-Length")
	if cl == "" {
		return -1
	}
	const maxSafe = (1<<63 - 1) / 10 // prevent overflow on multiply
	var n int
	for _, c := range cl {
		if c >= '0' && c <= '9' {
			if n > maxSafe {
				return -1 // overflow
			}
			n = n*10 + int(c-'0')
			if n < 0 {
				return -1 // overflow
			}
		} else {
			return -1
		}
	}
	return n
}

// Cookie returns a specific cookie by name, or nil if not found.
func (r *Response) Cookie(name string) *http.Cookie {
	for _, c := range r.Cookies {
		if c.Name == name {
			return c
		}
	}
	return nil
}
