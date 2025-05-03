package balancer

import (
	"net/url"
)

// ParseURL parses a URL string into a URL object
func ParseURL(rawURL string) (*url.URL, error) {
	return url.Parse(rawURL)
}
