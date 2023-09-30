// Package main provides the entry point for the application.
// It includes functions for loading and shuffling parameters and proxies,
// setting up loggers, progress bar, worker pool, and starting threads for sending requests.
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/vbauerster/mpb/v7"
	"github.com/vbauerster/mpb/v7/decor"
)

// Initialize counters
var successCount int32
var failureCount int32
var totalRequests int32
var successfulProxyConnections int32
var failedProxyConnections int32

// Proxies pool
var proxiesPool = make(chan string, numOfThreads)

// main is the entry point of the application. It loads and shuffles parameters and proxies,
// sets up loggers and the progress bar, starts threads for sending requests, and prints stats.
func main() {
	// Load and shuffle parameters and proxies
	if err := loadAndShuffleParametersAndProxies(); err != nil {
		log.Fatalf("Failed to load and shuffle parameters and proxies: %s", err)
	}

	// Get current directory
	dir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get current directory: %v", err)
	}

	// Construct log file paths
	logFilePath := filepath.Join(dir, logFileName)
	proxiesLogPath := filepath.Join(dir, proxiesLogName)

	// Setup loggers
	logFile, proxiesLogger, err := setupLoggers(logFilePath, proxiesLogPath)
	if err != nil {
		log.Fatalf("Failed to setup loggers: %s", err)
	}
	// Ensure logFile is closed properly
	defer func() {
		if err := logFile.Close(); err != nil {
			log.Printf("Failed to close log file: %s", err)
		}
	}()

	// Setup progress bar
	p, bar := setupProgressBar()

	// Start threads for sending requests
	if runIndefinitely {
		startThreadsIndefinitely(bar, proxiesLogger)
	} else {
		startThreads(bar, proxiesLogger)
	}

	// Print stats periodically
	printStats()

	// Wait for all progress bars to complete
	p.Wait()
}

// loadAndShuffleParametersAndProxies loads parameters and proxies from files and shuffles them.
// It returns an error if loading parameters or proxies fails.
func loadAndShuffleParametersAndProxies() error {
	// Load parameters
	if err := loadParameters(); err != nil {
		log.Printf("Error in loadAndShuffleParametersAndProxies: %v", err)
		return fmt.Errorf("Failed to load parameters: %w", err)
	}
	// Load proxies if useProxy is enabled
	if useProxy {
		if err := loadProxies(); err != nil {
			log.Printf("Error in loadAndShuffleParametersAndProxies: %v", err)
			return fmt.Errorf("Failed to load proxies: %w", err)
		}
	}

	// Shuffle proxies and parameters
	rand.Shuffle(len(proxies), func(i, j int) { proxies[i], proxies[j] = proxies[j], proxies[i] })
	rand.Shuffle(len(parameters), func(i, j int) { parameters[i], parameters[j] = parameters[j], parameters[i] })

	return nil
}

// setupLoggers sets up the main and proxies loggers.
// It returns the log file, the proxies logger, and an error if setting up loggers fails.
func setupLoggers(logFilePath string, proxiesLogPath string) (*os.File, *log.Logger, error) {
	// Set up logging to a file
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Printf("Error in setupLoggers: %v", err)
		return nil, nil, fmt.Errorf("Failed to open log file: %w", err)
	}
	log.SetOutput(logFile)

	// Set up logging for proxies to a separate file
	proxiesLogger, err := setupProxiesLogger(proxiesLogPath)
	if err != nil {
		log.Printf("Error in setupLoggers: %v", err)
		return nil, nil, fmt.Errorf("Failed to set up proxies logger: %w", err)
	}

	return logFile, proxiesLogger, nil
}

// setupProgressBar sets up the progress bar.
// It returns the progress object and the bar object.
func setupProgressBar() (*mpb.Progress, *mpb.Bar) {
	// Create a new progress bar with a large total
	p := mpb.New(mpb.WithWidth(60))
	var total int64
	if runIndefinitely {
		total = int64(math.MaxInt64)
	} else {
		total = int64(numOfThreads * numOfRequests)
	}
	bar := p.AddBar(total,
		mpb.PrependDecorators(
			decor.Name("Processing: ", decor.WCSyncSpace),
			decor.CountersNoUnit("%d / %d", decor.WCSyncWidth),
		),
		mpb.AppendDecorators(
			decor.Percentage(decor.WCSyncSpace),
		),
	)

	return p, bar
}

// Global variable for the total number of requests sent by all threads
var totalRequestCount int32

// worker is a goroutine that continuously creates and tests proxies.
func worker(proxiesLogger *log.Logger) {
	for {
		// Break the loop after all threads have obtained a proxy
		if atomic.LoadInt32(&successfulProxyConnections) >= numOfThreads {
			break
		}

		var proxy string
		if useProxy {
			for {
				proxy = proxies[rand.Intn(len(proxies))]

				// Check if the proxy IP is unique
				if _, exists := uniqueIPs.Load(proxy); !exists {
					// Test the proxy
					client, err := createProxyClient(proxy)
					if err != nil || !testProxy(client, proxiesLogger) {
						atomic.AddInt32(&failedProxyConnections, 1)
						continue
					}

					atomic.AddInt32(&successfulProxyConnections, 1)
					uniqueIPs.Store(proxy, true)
					break
				}
			}
		}
		proxiesPool <- proxy
	}
}

// thread is a goroutine that sends requests and calculates stats.
// It gets a unique proxy from the proxies pool, creates a client, sends requests, and then discards the client.
// It keeps sending requests indefinitely or until it fails.
func thread(bar *mpb.Bar, proxiesLogger *log.Logger) {
	for {
		// Get a unique proxy from the proxies pool
		proxy := <-proxiesPool

		// Create a client with the proxy
		client, err := createProxyClient(proxy)
		if err != nil {
			proxiesLogger.Printf("Failed to create client with proxy %s: %s\n", proxy, err)
			continue
		}

		summaries := make([]RequestSummary, 0)
		durations := make([]time.Duration, 0)
		sizes := make([]int, 0)

		requestCount := 0
		for {
			sendRequest(client, bar, &summaries, &durations, &sizes)
			requestCount++

			if requestCount >= numOfRequests {
				break
			}
		}

		if atomic.AddInt32(&totalRequestCount, int32(requestCount)) >= int32(numOfThreads*numOfRequests) {
			return
		}
	}
}

// threadIndefinitely is a goroutine that sends requests indefinitely and calculates stats.
// It gets a unique proxy from the proxies pool, creates a client, sends requests, and then returns the proxy to the pool.
func threadIndefinitely(bar *mpb.Bar, proxiesLogger *log.Logger) {
	for {
		// Get a unique proxy from the proxies pool
		proxy := <-proxiesPool

		// Create a client with the proxy
		client, err := createProxyClient(proxy)
		if err != nil {
			proxiesLogger.Printf("Failed to create client with proxy %s: %s\n", proxy, err)
			continue
		}

		summaries := make([]RequestSummary, 0)
		durations := make([]time.Duration, 0)
		sizes := make([]int, 0)

		requestCount := 0
		for {
			sendRequest(client, bar, &summaries, &durations, &sizes)
			requestCount++

			if requestCount >= numOfRequests {
				break
			}
		}

		// Return the proxy to the pool for reuse
		proxiesPool <- proxy
	}
}

// startThreads starts the threads for sending requests.
func startThreads(bar *mpb.Bar, proxiesLogger *log.Logger) {
	// Start the workers
	for i := 0; i < numOfThreads; i++ {
		go worker(proxiesLogger)
	}

	// Start the threads
	for i := 0; i < numOfThreads; i++ {
		go thread(bar, proxiesLogger)
	}
}

// startThreadsIndefinitely starts the threads for sending requests indefinitely.
func startThreadsIndefinitely(bar *mpb.Bar, proxiesLogger *log.Logger) {
	// Start the workers
	for i := 0; i < numOfThreads; i++ {
		go worker(proxiesLogger)
	}

	// Start the threads
	for {
		if atomic.LoadInt32(&successfulProxyConnections) >= numOfThreads {
			break
		}
		go threadIndefinitely(bar, proxiesLogger)
	}
}

// sendRequest sends a request, updates the stats and increments the progress bar.
func sendRequest(client *http.Client, bar *mpb.Bar, summaries *[]RequestSummary, durations *[]time.Duration, sizes *[]int) {
	// Select a random parameter and generate a unique random number for each request
	param := parameters[rand.Intn(len(parameters))] + "=" + rng()

	// Call onRequest function to increment the total requests and requests per minute counters
	onRequest()

	summary := RequestSummary{
		Parameter: param,
	}

	url := baseUrl + "?" + param

	// Create a new request
	ctx, cancel := context.WithTimeout(context.Background(), clientTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		log.Printf("Failed to create request with parameter %s: %s\n", param, err)
		atomic.AddInt32(&failureCount, 1)
		return
	}
	req.Header.Add("Accept-Language", language)
	req.Header.Add("Content-Type", contentType)
	// Send the request and measure the time it takes
	start := time.Now()
	resp, err := client.Do(req)
	if fireAndForget {
		bar.Increment() // Increment the progress bar
		return
	}
	duration := time.Since(start)
	summary.Duration = duration
	if err != nil {
		log.Printf("Failed on request with parameter %s: %s\n", param, err)
		summary.ErrorCount++
		atomic.AddInt32(&failureCount, 1)
		bar.Increment() // Increment the progress bar
		return
	}

	// Read the response body
	var body []byte
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read response body for request with parameter %s: %s\n", param, err)
		summary.ErrorCount++
		atomic.AddInt32(&failureCount, 1)
	} else {
		summary.BytesIn = len(body)
		*sizes = append(*sizes, len(body))
	}

	// Close the response body and handle any error
	if err := resp.Body.Close(); err != nil {
		log.Printf("Failed to close response body: %s", err)
	}

	// Append the duration and the summary to their respective slices
	*durations = append(*durations, duration)
	*summaries = append(*summaries, summary)

	log.Printf("Successful request with parameter %s: %d bytes, %s\n", param, len(body), duration)

	// Increment the success counter
	atomic.AddInt32(&successCount, 1)

	// Increment the progress bar
	bar.Increment()
}
