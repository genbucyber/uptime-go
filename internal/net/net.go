package net

import (
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/url"
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
		return nil, err
	}

	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	responseTime := time.Since(start)
	success := resp.StatusCode >= 200 && resp.StatusCode < 300
	isUp := success

	result := &CheckResults{
		URL:          nc.URL,
		LastCheck:    time.Now(),
		ResponseTime: responseTime,
		IsUp:         isUp,
		StatusCode:   resp.StatusCode,
		ErrorMessage: "",
	}

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

func GetIPAddress() (string, error) {
	resp, err := http.Get("https://api.ipify.org")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}
