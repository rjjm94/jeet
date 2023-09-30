// utils.go contains utility functions to load proxies from a file and test a proxy.

package main

import (
	"io"
	"log"
	"net/http"
)

// testProxy tests a proxy by sending a request to the test URL.
func testProxy(client *http.Client, proxiesLogger *log.Logger) bool {
	resp, err := client.Get(testUrl)
	if err != nil {
		proxiesLogger.Printf("Failed to connect to test URL with proxy: %s\n", err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		proxiesLogger.Printf("Received non-200 response code: %d\n", resp.StatusCode)
		return false
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		proxiesLogger.Printf("Failed to read response body: %s\n", err)
		return false
	}

	// Add the IP to uniqueIPs
	ip := string(body)
	uniqueIPs.Store(ip, true)

	return true
}
