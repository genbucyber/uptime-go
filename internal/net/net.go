package net

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"uptime-go/internal/incident"
	"uptime-go/internal/version"
)

const (
	ipTypeBoth = "both"
	ipTypeV4   = "ipv4"
	ipTypeV6   = "ipv6"
)

type NetworkConfig struct {
	URL             string
	RefreshInterval time.Duration
	Timeout         time.Duration
	FollowRedirects bool
	SkipSSL         bool
	IPType          string

	// Granular timeouts for different phases
	DNSTimeout            time.Duration
	DialTimeout           time.Duration
	TLSHandshakeTimeout   time.Duration
	ResponseHeaderTimeout time.Duration
}

type CheckResults struct {
	URL            string
	LastCheck      time.Time
	ResponseTime   time.Duration
	IsUp           bool
	StatusCode     int
	ContentSize	   int64
	Body		   string
	IncidentType   incident.Type
	ErrorMessage   string
	SSLExpiredDate *time.Time

	// Phase timing breakdown (for debugging)
	DNSTime       time.Duration
	ConnectTime   time.Duration
	TLSTime       time.Duration
	FirstByteTime time.Duration
}

func (nc *NetworkConfig) CheckWebsite() (*CheckResults, error) {
	result := &CheckResults{
		URL:       nc.URL,
		LastCheck: time.Now(),
		IsUp:      false,
	}

	// Set default granular timeouts if not specified
	dnsTimeout := nc.DNSTimeout
	if dnsTimeout == 0 {
		dnsTimeout = 5 * time.Second
	}

	dialTimeout := nc.DialTimeout
	if dialTimeout == 0 {
		dialTimeout = 10 * time.Second
	}

	tlsTimeout := nc.TLSHandshakeTimeout
	if tlsTimeout == 0 {
		tlsTimeout = 10 * time.Second
	}

	headerTimeout := nc.ResponseHeaderTimeout
	if headerTimeout == 0 {
		headerTimeout = 20 * time.Second
	}

	// Create context with overall timeout as safety net
	totalTimeout := nc.Timeout
	if totalTimeout == 0 {
		totalTimeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), totalTimeout)
	defer cancel()

	// Track timing for each phase
	var dnsStart, connectStart time.Time

	// Custom dialer with DNS and connection timeouts
	dialer := &net.Dialer{
		Timeout:   dialTimeout,
		KeepAlive: 30 * time.Second,
	}

	// Create transport with granular timeouts
	transport := &http.Transport{
		// Custom DialContext to track DNS and connection timing
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			// DNS resolution phase
			dnsStart = time.Now()

			// Create sub-context with DNS timeout
			dnsCtx, dnsCancel := context.WithTimeout(ctx, dnsTimeout)
			defer dnsCancel()

			// Resolve DNS
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, fmt.Errorf("invalid address: %w", err)
			}

			ipVersion := normalizeIPType(nc.IPType)
			lookupNetwork := "ip"
			if ipVersion == ipTypeV4 {
				lookupNetwork = "ip4"
			} else if ipVersion == ipTypeV6 {
				lookupNetwork = "ip6"
			}

			// Use "ip", "ip4", or "ip6" network type for DNS lookup
			ips, err := net.DefaultResolver.LookupIP(dnsCtx, lookupNetwork, host)
			if err != nil {
				return nil, fmt.Errorf("DNS resolution failed: %w", err)
			}
			result.DNSTime = time.Since(dnsStart)

			if len(ips) == 0 {
				switch ipVersion {
				case ipTypeV4:
					return nil, fmt.Errorf("no IPv4 addresses found for host: %s", host)
				case ipTypeV6:
					return nil, fmt.Errorf("no IPv6 addresses found for host: %s", host)
				default:
					return nil, fmt.Errorf("no IP addresses found for host: %s", host)
				}
			}

			// TCP connection phase
			connectStart = time.Now()
			ipAddr := net.JoinHostPort(ips[0].String(), port)
			conn, err := dialer.DialContext(ctx, network, ipAddr)
			if err != nil {
				return nil, fmt.Errorf("TCP connection failed: %w", err)
			}
			result.ConnectTime = time.Since(connectStart)

			return conn, nil
		},

		// TLS handshake timeout
		TLSHandshakeTimeout: tlsTimeout,

		// Response header timeout
		ResponseHeaderTimeout: headerTimeout,

		// Expect 100-continue timeout
		ExpectContinueTimeout: 1 * time.Second,

		// TLS configuration
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: nc.SkipSSL || isIPAddress(nc.URL),
		},

		// Connection pool settings
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableKeepAlives:  true, // Don't reuse connections for monitoring
		DisableCompression: false,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   totalTimeout,
	}

	// Handle redirects
	if !nc.FollowRedirects {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, nc.URL, nil)
	if err != nil {
		result.ErrorMessage = err.Error()
		return result, err
	}

	req.Header.Set("User-Agent", "GenbuUptimePlugin/"+version.VERSION)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Connection", "close")

	requestStart := time.Now()
	resp, err := client.Do(req)
	responseTime := time.Since(requestStart)
	result.ResponseTime = responseTime
	result.FirstByteTime = responseTime - result.DNSTime - result.ConnectTime

	if err != nil {
		result.ErrorMessage = nc.categorizeError(err, totalTimeout, dnsTimeout, dialTimeout, tlsTimeout, headerTimeout)
		if errors.Is(err, context.DeadlineExceeded) {
			result.IncidentType = incident.Timeout
		}else{
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() {
				result.IncidentType = incident.Timeout
			}else{
				result.IncidentType = incident.UnexpectedStatusCode
			}
		}
		return result, err
	}
	defer resp.Body.Close()

	var bodyBuf bytes.Buffer
	contentSize, err := io.Copy(&bodyBuf, resp.Body)

	if err != nil {
		result.IsUp = false
		result.ErrorMessage = nc.categorizeError(err, totalTimeout, dnsTimeout, dialTimeout, tlsTimeout, headerTimeout)
		if errors.Is(err, context.DeadlineExceeded) {
			result.IncidentType = incident.Timeout
		}else{
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() {
				result.IncidentType = incident.Timeout
			}else{
				result.IncidentType = incident.UnexpectedStatusCode
			}
		}
		return result, err
	}
	
	result.ContentSize = contentSize
	result.Body = bodyBuf.String()


	// Treat redirects (3xx) as UP so 302 doesn't mark the monitor down.
	success := resp.StatusCode >= 200 && resp.StatusCode < 400
	result.IsUp = success
	result.StatusCode = resp.StatusCode

	if !success {
		result.IncidentType = incident.UnexpectedStatusCode
		result.ErrorMessage = fmt.Sprintf("Received non-successful status code: %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	// Extract TLS information
	if resp.TLS != nil && len(resp.TLS.PeerCertificates) > 0 {
		result.SSLExpiredDate = &resp.TLS.PeerCertificates[0].NotAfter
	}

	return result, nil
}

func (nc *NetworkConfig) CategorizeError(err error) string {
	totalTimeout, dnsTimeout, dialTimeout, tlsTimeout, headerTimeout := nc.timeoutDefaults()
	return nc.categorizeError(err, totalTimeout, dnsTimeout, dialTimeout, tlsTimeout, headerTimeout)
}

func (nc *NetworkConfig) timeoutDefaults() (time.Duration, time.Duration, time.Duration, time.Duration, time.Duration) {
	dnsTimeout := nc.DNSTimeout
	if dnsTimeout == 0 {
		dnsTimeout = 5 * time.Second
	}

	dialTimeout := nc.DialTimeout
	if dialTimeout == 0 {
		dialTimeout = 10 * time.Second
	}

	tlsTimeout := nc.TLSHandshakeTimeout
	if tlsTimeout == 0 {
		tlsTimeout = 10 * time.Second
	}

	headerTimeout := nc.ResponseHeaderTimeout
	if headerTimeout == 0 {
		headerTimeout = 20 * time.Second
	}

	totalTimeout := nc.Timeout
	if totalTimeout == 0 {
		totalTimeout = 30 * time.Second
	}

	return totalTimeout, dnsTimeout, dialTimeout, tlsTimeout, headerTimeout
}

// categorizeError provides more detailed error messages based on the type of failure
func (nc *NetworkConfig) categorizeError(err error, totalTimeout, dnsTimeout, dialTimeout, tlsTimeout, headerTimeout time.Duration) string {
	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Sprintf("Overall request timeout after %v for %s. The site did not complete the check within response_time_threshold.", totalTimeout, nc.URL)
	}

	var opErr *net.OpError
	if errors.As(err, &opErr) {
		var dnsErr *net.DNSError
		if errors.As(opErr.Err, &dnsErr) {
			if dnsErr.Timeout() {
				return fmt.Sprintf("DNS resolution timeout after %v for %s. Check DNS records/resolver or increase dns_timeout.", dnsTimeout, nc.URL)
			}
			return fmt.Sprintf("DNS resolution failed for %s: %v", nc.URL, dnsErr)
		}

		if opErr.Timeout() {
			switch opErr.Op {
			case "dial":
				return fmt.Sprintf("TCP connection timeout after %v for %s. Check firewall, routing, server reachability, or increase dial_timeout.", dialTimeout, nc.URL)
			case "read":
				return fmt.Sprintf("Response timeout after %v for %s. The server accepted the connection but did not send a response in time; check application latency or increase response_header_timeout.", headerTimeout, nc.URL)
			default:
				return fmt.Sprintf("Network timeout during %s for %s. The request exceeded a configured network timeout.", opErr.Op, nc.URL)
			}
		}

		return fmt.Sprintf("Network operation error for %s: %s - %v", nc.URL, opErr.Op, opErr.Err)
	}

	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return fmt.Sprintf("Connection closed prematurely (EOF) while fetching %s", nc.URL)
	}

	var tlsErr tls.RecordHeaderError
	if errors.As(err, &tlsErr) {
		return fmt.Sprintf("TLS handshake failed for %s: invalid record header", nc.URL)
	}

	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		if urlErr.Timeout() {
			if strings.Contains(strings.ToLower(urlErr.Err.Error()), "tls handshake timeout") {
				return fmt.Sprintf("TLS handshake timeout after %v for %s. Check TLS/certificate configuration or increase tls_handshake_timeout.", tlsTimeout, nc.URL)
			}
			return fmt.Sprintf("HTTP request timeout for %s. The request exceeded a configured timeout before a complete response was received.", nc.URL)
		}
		return fmt.Sprintf("URL error for %s: %v", nc.URL, urlErr.Err)
	}

	return fmt.Sprintf("Failed to fetch %s: %v", nc.URL, err)
}

func isIPAddress(host string) bool {
	u, err := url.Parse(host)
	if err != nil {
		return false
	}
	hostname := u.Hostname()

	return net.ParseIP(hostname) != nil
}

func normalizeIPType(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "":
		return ipTypeV4
	case ipTypeBoth:
		return ipTypeBoth
	case ipTypeV4:
		return ipTypeV4
	case ipTypeV6:
		return ipTypeV6
	default:
		return ipTypeV4
	}
}

var (
	ipAddress string
	once      sync.Once
)

func GetIPAddress() (string, error) {
	var err error
	once.Do(func() {
		urls := []string{
			"https://api.ipify.org",
			"https://ifconfig.me/ip",
		}

		for _, url := range urls {
			ipAddress, err = fetchIP(url)
			if err == nil {
				return
			}
		}
	})

	return ipAddress, err
}

func fetchIP(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}
