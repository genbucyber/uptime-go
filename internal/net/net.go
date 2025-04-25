package net

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
	"time"
	"uptime-go/internal/net/config"
)

type NetworkConfig struct {
	URL             string
	RefreshInterval time.Duration
	Timeout         time.Duration
	FollowRedirects bool
	SkipSSL         bool
}

func (nc *NetworkConfig) CheckWebsite() (*config.CheckResults, error) {
	client := &http.Client{
		Timeout: nc.Timeout,

		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: nc.SkipSSL || isIPAddress(nc.URL)},
		},
	}

	if !nc.FollowRedirects {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

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

	return &config.CheckResults{
		URL:          nc.URL,
		LastCheck:    time.Now(),
		ResponseTime: responseTime,
		IsUp:         isUp,
		StatusCode:   resp.StatusCode,
		ErrorMessage: "",
	}, nil
}

func isIPAddress(host string) bool {
	u, err := url.Parse(host)
	if err != nil {
		return false
	}
	hostname := u.Hostname()

	return net.ParseIP(hostname) != nil
}
