package net

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestCheckWebsiteErrorMessages(t *testing.T) {
	tests := []struct {
		name           string
		handler        http.HandlerFunc
		expectedErrMsg string
		expectErr      bool
	}{
		{
			name: "EOF error message",
			handler: func(w http.ResponseWriter, r *http.Request) {
				// Simulate an EOF by closing the connection prematurely
				hj, ok := w.(http.Hijacker)
				if !ok {
					http.Error(w, "webserver doesn't support hijacking", http.StatusInternalServerError)
					return
				}
				conn, _, err := hj.Hijack()
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				conn.Close()
			},
			expectedErrMsg: "Connection closed prematurely (EOF)",
			expectErr:      true,
		},
		{
			name:           "Connection refused error message",
			handler:        func(w http.ResponseWriter, r *http.Request) {},
			expectedErrMsg: "Network operation error for",
			expectErr:      true,
		},
		{
			name: "Generic error message",
			handler: func(w http.ResponseWriter, r *http.Request) {
				// Simulate a generic error
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("invalid response"))
			},
			expectedErrMsg: "Failed to fetch",
			expectErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var server *httptest.Server
			var targetURL string

			if tt.name == "Connection refused error message" {
				targetURL = "http://localhost:9999"
			} else {
				server = httptest.NewServer(tt.handler)
				defer server.Close()
				targetURL = server.URL
			}

			nc := NetworkConfig{
				URL:     targetURL,
				Timeout: 100 * time.Millisecond,
			}

			results, err := nc.CheckWebsite()

			if tt.expectErr {
				if err == nil {
					t.Fatalf("Expected an error, but got none")
				}
				if results == nil {
					t.Fatalf("Expected results, but got nil")
				}
				if !strings.Contains(results.ErrorMessage, tt.expectedErrMsg) {
					t.Errorf("Expected error message to contain '%s', but got '%s'", tt.expectedErrMsg, results.ErrorMessage)
				}
			} else {
				if err != nil {
					t.Fatalf("Did not expect an error, but got: %v", err)
				}
				if results == nil {
					t.Fatalf("Expected results, but got nil")
				}

				if strings.Contains(results.ErrorMessage, tt.expectedErrMsg) {
					t.Errorf("Did not expect error message to contain '%s', but got '%s'", tt.expectedErrMsg, results.ErrorMessage)
				}
			}
		})
	}
}

func TestCategorizeErrorUsesClearTimeoutMessages(t *testing.T) {
	nc := NetworkConfig{
		URL:                   "https://example.com",
		Timeout:               30 * time.Second,
		DNSTimeout:            5 * time.Second,
		DialTimeout:           10 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 20 * time.Second,
	}

	tests := []struct {
		name         string
		err          error
		wantContains []string
	}{
		{
			name: "overall timeout",
			err:  context.DeadlineExceeded,
			wantContains: []string{
				"Overall request timeout after 30s for https://example.com",
				"response_time_threshold",
			},
		},
		{
			name: "url timeout fallback",
			err:  &url.Error{Op: "Get", URL: "https://example.com", Err: context.DeadlineExceeded},
			wantContains: []string{
				"Overall request timeout after 30s for https://example.com",
				"response_time_threshold",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := nc.CategorizeError(tt.err)
			for _, want := range tt.wantContains {
				if !strings.Contains(msg, want) {
					t.Fatalf("expected %q to contain %q", msg, want)
				}
			}
			if strings.Contains(msg, "URL request timeout") || strings.Contains(msg, "Request timed out:") {
				t.Fatalf("expected clear timeout message, got %q", msg)
			}
		})
	}
}

func TestCheckWebsiteTimeoutMessageIsActionable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	nc := NetworkConfig{
		URL:     server.URL,
		Timeout: 10 * time.Millisecond,
	}

	results, err := nc.CheckWebsite()
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if results == nil {
		t.Fatal("expected results")
	}
	if !strings.Contains(results.ErrorMessage, "Overall request timeout after 10ms") {
		t.Fatalf("expected overall timeout message, got %q", results.ErrorMessage)
	}
	if !strings.Contains(results.ErrorMessage, "response_time_threshold") {
		t.Fatalf("expected actionable config hint, got %q", results.ErrorMessage)
	}
}

func TestCheckWebsiteSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}))
	defer server.Close()

	nc := NetworkConfig{
		URL:     server.URL,
		Timeout: 10 * time.Second,
	}

	results, err := nc.CheckWebsite()

	if err != nil {
		t.Fatalf("Expected no error, but got %v", err)
	}
	if results == nil {
		t.Fatalf("Expected results, but got nil")
	}
	if !results.IsUp {
		t.Errorf("Expected IsUp to be true, but got false")
	}
	if results.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, but got %d", http.StatusOK, results.StatusCode)
	}
	if results.ErrorMessage != "" {
		t.Errorf("Expected empty error message, but got %s", results.ErrorMessage)
	}
}

func TestCheckWebsiteReadBody(t *testing.T) {
	const forbiddenPhrase = "forbidden phrase after limit"
	body := strings.Repeat("a", 1*1024*1024) + forbiddenPhrase

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()

	results, err := (&NetworkConfig{
		URL:     server.URL,
		Timeout: 10 * time.Second,
	}).CheckWebsite()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if results.ContentSize != int64(len(body)) {
		t.Fatalf("expected content size %d, got %d", len(body), results.ContentSize)
	}
	if !strings.Contains(results.Body, forbiddenPhrase) {
		t.Fatal("expected response body past 1 MiB to be available for word validation")
	}
}

func TestCheckWebsiteIPv4Only(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen on IPv4: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("OK"))
		}),
	}
	go func() {
		_ = server.Serve(listener)
	}()
	t.Cleanup(func() { _ = server.Close() })

	url := "http://" + listener.Addr().String()
	nc := NetworkConfig{
		URL:     url,
		Timeout: 5 * time.Second,
		IPType:  "ipv4",
	}

	results, err := nc.CheckWebsite()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if results == nil || !results.IsUp {
		t.Fatalf("expected IPv4 check to be up, got %+v", results)
	}
}

func TestCheckWebsiteIPv6Only(t *testing.T) {
	listener, err := net.Listen("tcp6", "[::1]:0")
	if err != nil {
		t.Skipf("IPv6 not available: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("OK"))
		}),
	}
	go func() {
		_ = server.Serve(listener)
	}()
	t.Cleanup(func() { _ = server.Close() })

	addr := listener.Addr().(*net.TCPAddr)
	url := fmt.Sprintf("http://[%s]:%d", addr.IP.String(), addr.Port)
	nc := NetworkConfig{
		URL:     url,
		Timeout: 5 * time.Second,
		IPType:  "ipv6",
	}

	results, err := nc.CheckWebsite()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if results == nil || !results.IsUp {
		t.Fatalf("expected IPv6 check to be up, got %+v", results)
	}
}

func TestCheckWebsiteTruncatedBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "100")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("partial body"))
		if hj, ok := w.(http.Hijacker); ok {
			if con, _, err := hj.Hijack(); err == nil {
				_ = con.Close()
			}
		}
	}))
	defer server.Close()

	nc := NetworkConfig{
		URL: server.URL,
		Timeout: 3 * time.Second,
	}

	results, err := nc.CheckWebsite()
	if err == nil {
		t.Fatal("expected error for truncated body transfer, but got nil")
	}
	if results == nil {
		t.Fatal("expected results, but got nil")
	}
	if results.IsUp {
		t.Errorf("expected IsUp to be false for truncated body, got true")
	}
}