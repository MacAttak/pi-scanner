package test

import (
	"testing"
	"time"
)

// RetryConfig configures test retry behavior
type RetryConfig struct {
	MaxAttempts int
	Delay       time.Duration
	Backoff     float64 // Multiplier for delay between attempts
}

// DefaultRetryConfig returns a sensible default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts: 3,
		Delay:       2 * time.Second,
		Backoff:     1.5,
	}
}

// RetryTestFunc represents a test function that can be retried
type RetryTestFunc func() error

// WithRetry executes a test function with retry logic
func WithRetry(t *testing.T, testName string, config RetryConfig, testFunc RetryTestFunc) {
	t.Helper()

	var lastErr error
	delay := config.Delay

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		err := testFunc()
		if err == nil {
			// Test passed
			return
		}

		lastErr = err

		if attempt < config.MaxAttempts {
			t.Logf("%s: Attempt %d failed: %v, retrying in %v...",
				testName, attempt, err, delay)
			time.Sleep(delay)
			delay = time.Duration(float64(delay) * config.Backoff)
		} else {
			t.Logf("%s: All %d attempts failed", testName, config.MaxAttempts)
		}
	}

	// All attempts failed
	t.Errorf("%s failed after %d attempts. Last error: %v",
		testName, config.MaxAttempts, lastErr)
}

// WithSimpleRetry provides a simpler interface for basic retry scenarios
func WithSimpleRetry(t *testing.T, testName string, testFunc RetryTestFunc) {
	WithRetry(t, testName, DefaultRetryConfig(), testFunc)
}

// FlakeResistant marks a test as potentially flaky and applies retry logic
func FlakeResistant(t *testing.T, testFunc func(t *testing.T)) {
	t.Helper()

	// Use a more aggressive retry strategy for flaky tests
	config := RetryConfig{
		MaxAttempts: 5,
		Delay:       1 * time.Second,
		Backoff:     1.2,
	}

	WithRetry(t, t.Name(), config, func() error {
		// Create a sub-test to capture failures
		var failed bool
		t.Run("attempt", func(subT *testing.T) {
			// Recover from any panics to convert them to errors
			defer func() {
				if r := recover(); r != nil {
					failed = true
					subT.Errorf("test panicked: %v", r)
				}
			}()

			testFunc(subT)
			if subT.Failed() {
				failed = true
			}
		})

		if failed {
			return &testError{message: "test failed"}
		}
		return nil
	})
}

// testError represents a test error
type testError struct {
	message string
}

func (e *testError) Error() string {
	return e.message
}

// NetworkTest marks a test as requiring network access and applies appropriate handling
func NetworkTest(t *testing.T, testFunc func(t *testing.T)) {
	t.Helper()

	// Skip network tests if explicitly disabled
	if getBoolEnvOrDefault("SKIP_NETWORK_TESTS", false) {
		t.Skip("Network tests disabled")
	}

	// Apply retry logic for network tests
	FlakeResistant(t, testFunc)
}

// AuthTest marks a test as requiring authentication and applies appropriate handling
func AuthTest(t *testing.T, testFunc func(t *testing.T)) {
	t.Helper()

	// Skip auth tests if explicitly disabled
	if getBoolEnvOrDefault("SKIP_AUTH_TESTS", false) {
		t.Skip("Authentication tests disabled")
	}

	// Check for available authentication
	if !hasAuthentication() {
		t.Skip("No authentication available")
	}

	// Apply retry logic for auth tests
	FlakeResistant(t, testFunc)
}

// CITest marks a test as CI-specific and applies appropriate handling
func CITest(t *testing.T, testFunc func(t *testing.T)) {
	t.Helper()

	// Only run in CI environments
	if !isCIEnvironment() {
		t.Skip("CI-only test")
	}

	// CI tests get more aggressive timeouts and retries
	config := RetryConfig{
		MaxAttempts: 2, // Shorter retry in CI to avoid timeouts
		Delay:       1 * time.Second,
		Backoff:     1.0, // No backoff in CI
	}

	WithRetry(t, t.Name(), config, func() error {
		// Create a sub-test to capture failures
		var failed bool
		t.Run("ci_attempt", func(subT *testing.T) {
			// Recover from any panics to convert them to errors
			defer func() {
				if r := recover(); r != nil {
					failed = true
					subT.Errorf("CI test panicked: %v", r)
				}
			}()

			testFunc(subT)
			if subT.Failed() {
				failed = true
			}
		})

		if failed {
			return &testError{message: "CI test failed"}
		}
		return nil
	})
}

// Helper functions

func getBoolEnvOrDefault(key string, defaultValue bool) bool {
	config := GetTestConfig()
	switch key {
	case "SKIP_NETWORK_TESTS":
		return config.SkipNetworkTests
	case "SKIP_AUTH_TESTS":
		return config.SkipAuthTests
	default:
		return defaultValue
	}
}

func hasAuthentication() bool {
	config := GetTestConfig()
	return config.GitHubToken != "" || isCIEnvironment()
}

func isCIEnvironment() bool {
	return isCI() // Use the function from test_config.go
}
