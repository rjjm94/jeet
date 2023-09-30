// stats.go contains the function to print statistics about the requests.

package main

import (
	"fmt"
	"sync/atomic"
	"time"
)

// requestPerMinute is a counter for the number of requests in the current minute
var requestPerMinute int32

// printStats prints statistics about the requests every second.
// It prints the total number of requests, success count, failure count,
// successful proxy connections, failed proxy connections, unique IPs, and requests per minute.
// The function does not take any arguments and does not return anything.
func printStats() {
	go func() {
		// Create a ticker that ticks every second
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		// Create a ticker that ticks every minute
		minuteTicker := time.NewTicker(1 * time.Minute)
		defer minuteTicker.Stop()

		for {
			select {
			case <-ticker.C:
				// Every second, print the statistics
				// Count the number of unique IPs
				uniqueIPCount := 0
				uniqueIPs.Range(func(key, value interface{}) bool {
					uniqueIPCount++
					return true
				})

				// Print the statistics
				fmt.Printf("\n--- STATS ---\n")
				fmt.Printf("Total requests: %d\n", atomic.LoadInt32(&totalRequests))
				fmt.Printf("Success count: %d\n", atomic.LoadInt32(&successCount))
				fmt.Printf("Failure count: %d\n", atomic.LoadInt32(&failureCount))
				fmt.Printf("Successful proxy connections: %d\n", atomic.LoadInt32(&successfulProxyConnections))
				fmt.Printf("Failed proxy connections: %d\n", atomic.LoadInt32(&failedProxyConnections))
				fmt.Printf("Unique IPs: %d\n", uniqueIPCount)
				fmt.Printf("Requests per minute: %d\n", atomic.LoadInt32(&requestPerMinute))
				fmt.Printf("-------------\n")
			case <-minuteTicker.C:
				// Every minute, reset the requests per minute counter
				atomic.StoreInt32(&requestPerMinute, 0)
			}
		}
	}()
}

// When a request is made, increment the total requests and requests per minute counters
func onRequest() {
	atomic.AddInt32(&totalRequests, 1)
	atomic.AddInt32(&requestPerMinute, 1)
}
