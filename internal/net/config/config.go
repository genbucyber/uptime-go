package config

import (
	"time"
)

type NetworkConfig struct {
	URL             string        // URL to check
	RefreshInterval time.Duration // Interval between checks (for monitoring mode)
	Timeout         time.Duration // HTTP request timeout
	FollowRedirects bool          // Whether to follow HTTP redirects
	SkipSSL         bool          // Whether to skip SSL certificate verification
}

type CheckResults struct {
	URL          string
	LastCheck    time.Time
	ResponseTime time.Duration
	IsUp         bool
	StatusCode   int
	ErrorMessage string
}
