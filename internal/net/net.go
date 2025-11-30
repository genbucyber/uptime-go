package net

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type NetworkConfig struct {
	URL             string
	RefreshInterval time.Duration
	Timeout         time.Duration
	FollowRedirects bool
	SkipSSL         bool
}

type CheckResults struct {
	URL            string
	LastCheck      time.Time
	ResponseTime   time.Duration
	IsUp           bool
	StatusCode     int
	ErrorMessage   string
	SSLExpiredDate *time.Time
}

func (nc *NetworkConfig) CheckWebsite() (*CheckResults, error) {
	result := &CheckResults{
		URL:       nc.URL,
		LastCheck: time.Now(),
		IsUp:      false,
	}

	client := &http.Client{
		Timeout: nc.Timeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: nc.SkipSSL || isIPAddress(nc.URL)},
		},
	}

	// TODO: later
	// if !nc.FollowRedirects {
	// 	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
	// 		return http.ErrUseLastResponse
	// 	}
	// }

	req, err := http.NewRequest(http.MethodGet, nc.URL, nil)
	if err != nil {
		result.ErrorMessage = err.Error()
		return result, err
	}

	req.Header.Set("User-Agent", "GenbuUptimePlugin/0.2")

	start := time.Now()
	resp, err := client.Do(req)
	responseTime := time.Since(start)
	result.ResponseTime = responseTime

	if err != nil {
		var opErr *net.OpError
		if errors.Is(err, io.EOF) {
			result.ErrorMessage = fmt.Sprintf("Connection closed prematurely (EOF) while fetching %s. This might indicate a server issue or an incomplete response.", nc.URL)
		} else if errors.As(err, &opErr) {
			result.ErrorMessage = fmt.Sprintf("Network operation error for %s: %s. Check connectivity or target server status. Original error: %v", nc.URL, opErr.Op, opErr.Err)
		} else {
			result.ErrorMessage = fmt.Sprintf("Failed to fetch %s: %v", nc.URL, err)
		}
		return result, err
	}
	defer resp.Body.Close()

	success := resp.StatusCode >= 200 && resp.StatusCode < 300
	result.IsUp = success
	result.StatusCode = resp.StatusCode

	if tls := resp.TLS; tls != nil &&
		tls.PeerCertificates != nil &&
		len(tls.PeerCertificates) != 0 {
		result.SSLExpiredDate = &tls.PeerCertificates[0].NotAfter
		// fmt.Printf("TLS: %v\n", time.Until(resp.TLS.PeerCertificates[0].NotAfter))
	}

	return result, nil
}

func isIPAddress(host string) bool {
	u, err := url.Parse(host)
	if err != nil {
		return false
	}
	hostname := u.Hostname()

	return net.ParseIP(hostname) != nil
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
