// client.go contains the function to create a new HTTP client with proxy support.

package main

import (
	"context"
	"fmt"
	"golang.org/x/net/proxy"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
)

// httpClientPool is a channel that holds HTTP clients.
// It has a capacity of numOfThreads.
var httpClientPool = make(chan *http.Client, numOfThreads)

// createProxyClient creates a new HTTP client with proxy support.
// It tries to create a client with the given proxy URL.
// If it fails, it retries up to retryCount times.
// If it succeeds, it adds the client to the HTTP client pool.
// If there is a client available in the pool, it returns that client.
// If there is no client available in the pool, it creates a new client.
// The function takes a string argument proxyURL which is the URL of the proxy to use.
// It returns a pointer to an http.Client and an error.
func createProxyClient(proxyURL string) (*http.Client, error) {
	// If there is a client available in the pool, return it
	select {
	case client := <-httpClientPool:
		return client, nil
	default:
	}

	// If the proxy URL does not start with "socks5://", add it
	if !strings.HasPrefix(proxyURL, "socks5://") {
		proxyURL = "socks5://" + proxyURL
	}

	// Parse the proxy URL
	u, err := url.Parse(proxyURL)
	if err != nil {
		log.Printf("Error in createProxyClient: %v", err)
		return nil, fmt.Errorf("Failed to parse proxy URL: %w", err)
	}

	// If the proxy URL has a user, create an Auth structure
	var auth *proxy.Auth
	if u.User != nil {
		password, _ := u.User.Password()
		auth = &proxy.Auth{
			User:     u.User.Username(),
			Password: password,
		}
	}

	// Try to create a dialer up to retryCount times
	var dialer proxy.Dialer
	for i := 0; i < retryCount; i++ {
		dialer, err = proxy.SOCKS5("tcp", u.Host, auth, proxy.Direct)
		if err == nil {
			break
		}
	}
	if err != nil {
		log.Printf("Error in createProxyClient: %v", err)
		return nil, fmt.Errorf("Failed to create dialer after %d attempts: %w", retryCount, err)
	}

	// Create an HTTP transport with the dialer
	httpTransport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.Dial(network, addr)
		},
		ForceAttemptHTTP2:     forceAttemptHTTP2,
		MaxIdleConns:          maxIdleConns,
		IdleConnTimeout:       idleConnTimeout,
		TLSHandshakeTimeout:   tlsHandshakeTimeout,
		ExpectContinueTimeout: expectContinueTimeout,
	}

	// Create an HTTP client with the transport
	client := &http.Client{
		Transport: httpTransport,
		Timeout:   clientTimeout,
	}

	// Add the client to the pool
	httpClientPool <- client

	return client, nil
}
