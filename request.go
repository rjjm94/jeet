// request.go contains the definitions for RequestSummary and ParameterSummary structs,
// and functions to generate random numbers and load parameters from a file.

package main

import (
	"bufio"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sync"
	"time"
)

// RequestSummary represents the summary of a request.
type RequestSummary struct {
	Parameter  string
	BytesIn    int
	Duration   time.Duration
	ErrorCount int
}

// ParameterSummary represents the summary of a parameter.
type ParameterSummary struct {
	Parameter    string
	MeanDuration time.Duration
	MeanSize     int
}

// rng generates a random number as a string.
func rng(args ...int) string {
	var min int
	var max int

	if len(args) > 1 {
		min = args[0]
		max = args[1]
	} else {
		min = 0
		max = 1000000
	}

	return fmt.Sprintf("%d", rand.Intn(max-min+1)+min)
}

// loadProxies loads the proxies from the proxies file in parallel.
// It reads the proxies from a file and sends them to a channel.
// Another goroutine receives the proxies from the channel and adds them to the proxies slice.
// If no proxies are found in the file, it returns an error.
func loadProxies() error {
	// Open the proxies file
	file, err := os.Open(proxiesFile)
	if err != nil {
		log.Printf("Error in loadProxies: %v", err)
		return fmt.Errorf("Failed to open proxies file: %w", err)
	}
	// Ensure the file is closed after the function returns
	defer func() {
		if cerr := file.Close(); cerr != nil {
			log.Printf("Failed to close proxies file: %s", cerr)
		}
	}()

	// Create a channel to send proxies
	proxyChan := make(chan string)

	wg := sync.WaitGroup{}
	wg.Add(2) // There are 2 goroutines to wait for

	// Start a goroutine to read proxies from the file and send them to the channel
	go func() {
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			proxyChan <- scanner.Text()
		}
		close(proxyChan)
		wg.Done() // This goroutine is done
	}()

	// Start another goroutine to receive proxies from the channel and add them to the proxies slice
	go func() {
		for proxy := range proxyChan {
			proxies = append(proxies, proxy)
		}
		wg.Done() // This goroutine is done
	}()

	wg.Wait() // Wait for all goroutines to finish

	// If no proxies were found in the file, return an error
	if len(proxies) == 0 {
		log.Printf("Error in loadProxies: No proxies found in the file")
		return fmt.Errorf("No proxies found in the file")
	}

	return nil
}

// loadParameters loads parameters from a file and appends them to the parameters slice.
func loadParameters() error {
	file, err := os.Open(parametersFile)
	if err != nil {
		log.Printf("Error in loadParameters: %v", err)
		return fmt.Errorf("Failed to open parameters file: %w", err)
	}
	// Defer file.Close() with error handling
	defer func() {
		if cerr := file.Close(); cerr != nil {
			log.Printf("Failed to close parameters file: %s", cerr)
		}
	}()

	params := make(chan string)

	wg := sync.WaitGroup{}
	wg.Add(2) // There are 2 goroutines to wait for

	// Read parameters from the file and send them to the params channel
	go func() {
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			params <- scanner.Text()
		}
		close(params)
		wg.Done() // This goroutine is done
	}()

	// Receive parameters from the params channel and append them to the parameters slice
	go func() {
		for param := range params {
			parameters = append(parameters, param)
		}
		wg.Done() // This goroutine is done
	}()

	wg.Wait() // Wait for all goroutines to finish

	if len(parameters) == 0 {
		log.Printf("Error in loadParameters: No parameters found in the file")
		return fmt.Errorf("No parameters found in the file")
	}

	return nil
}
