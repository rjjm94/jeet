// config.go contains the constants and global variables used throughout the application.

package main

import (
	"sync"
	"time"
)

// Constants for the application
const (
	baseUrl         = "https://thornode.ninerealms.com/thorchain/pool/BTC.BTC/liquidity_providers?height=%rng(12450000,12810000)" // Base URL for the requests
	clientTimeout   = 10 * time.Second                                                                                            // HTTP client timeout
	numOfThreads    = 500                                                                                                         // Number of threads to use
	numOfRequests   = 10                                                                                                          // Number of requests per thread
	retryCount      = 3                                                                                                           // Number of times to retry failed requests
	logFileName     = "requests.log"                                                                                              // Name of the log file
	proxiesLogName  = "proxies.log"                                                                                               // Name of the proxies log file
	language        = "EL"                                                                                                        // Accept-Language header value
	contentType     = "application/xml"                                                                                           // Content-Type header value
	parametersFile  = "parameters.txt"                                                                                            // File containing the parameters for the requests
	proxiesFile     = "proxy.txt"                                                                                                 // File containing the proxies
	runIndefinitely = false                                                                                                       // Whether to run indefinitely
	fireAndForget   = false                                                                                                       // Whether to send the request and hang up on the response
	useProxy        = true                                                                                                        // Whether to use proxies
	testUrl         = "http://api.ipify.org"                                                                                      // Test URL for testing proxies

	forceAttemptHTTP2     = false            // Whether to force HTTP/2 for the HTTP transport
	maxIdleConns          = 100              // Maximum number of idle connections for the HTTP transport
	idleConnTimeout       = 90 * time.Second // Idle connection timeout for the HTTP transport
	tlsHandshakeTimeout   = 10 * time.Second // TLS handshake timeout for the HTTP transport
	expectContinueTimeout = 1 * time.Second  // Expect-continue timeout for the HTTP transport
)

// Global variables for the application
var (
	parameters []string // Parameters for the requests
	proxies    []string // Proxies to use
	uniqueIPs  sync.Map // Unique IPs, used to keep track of unique IP addresses
)
