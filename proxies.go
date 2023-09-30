// proxies.go contains the function to set up logging for proxies to a separate file.

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// setupProxiesLogger sets up logging for proxies to a separate file.
// It checks if the proxies log file path is valid and absolute.
// If the path is not valid or not absolute, it returns an error.
// If the path is valid and absolute, it opens or creates the proxies log file.
// If it fails to open or create the file, it returns an error.
// If it succeeds in opening or creating the file, it creates a new logger for proxies and returns the logger.
// The function does not take any arguments and returns a pointer to a log.Logger and an error.
func setupProxiesLogger(proxiesLogPath string) (*log.Logger, error) {
	// Check if proxies log file path is valid
	if !filepath.IsAbs(proxiesLogPath) {
		log.Printf("Error in setupProxiesLogger: proxies log file path is not an absolute path: %s", proxiesLogPath)
		return nil, fmt.Errorf("proxies log file path is not an absolute path: %s", proxiesLogPath)
	}

	// Open or create the proxies log file
	proxiesLogFile, err := os.OpenFile(proxiesLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		// Distinguish between different kinds of errors for better error handling
		if os.IsPermission(err) {
			log.Printf("Error in setupProxiesLogger: permission denied while trying to open proxies log file: %v", err)
			return nil, fmt.Errorf("permission denied while trying to open proxies log file: %w", err)
		} else if os.IsNotExist(err) {
			log.Printf("Error in setupProxiesLogger: proxies log file does not exist: %v", err)
			return nil, fmt.Errorf("proxies log file does not exist: %w", err)
		} else {
			log.Printf("Error in setupProxiesLogger: failed to open proxies log file: %v", err)
			return nil, fmt.Errorf("failed to open proxies log file: %w", err)
		}
	}

	// Create a new logger for proxies
	proxiesLogger := log.New(proxiesLogFile, "", log.LstdFlags)

	return proxiesLogger, nil
}
