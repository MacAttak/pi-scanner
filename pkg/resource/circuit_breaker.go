package resource

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// CircuitState represents the state of a circuit breaker
type CircuitState int

const (
	// StateClosed allows requests to pass through
	StateClosed CircuitState = iota
	// StateOpen blocks all requests
	StateOpen
	// StateHalfOpen allows limited requests to test recovery
	StateHalfOpen
)

// CircuitBreaker protects resources from cascading failures
type CircuitBreaker struct {
	name            string
	state           CircuitState
	failures        int
	successes       int
	lastFailureTime time.Time
	mu              sync.RWMutex
	logger          *slog.Logger

	// Configuration
	maxFailures      int
	resetTimeout     time.Duration
	halfOpenRequests int
	halfOpenSuccess  int
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(name string, maxFailures int, resetTimeout time.Duration, logger *slog.Logger) *CircuitBreaker {
	return &CircuitBreaker{
		name:             name,
		state:            StateClosed,
		maxFailures:      maxFailures,
		resetTimeout:     resetTimeout,
		halfOpenRequests: 3,
		halfOpenSuccess:  2,
		logger:           logger,
	}
}

// Call executes the function with circuit breaker protection
func (cb *CircuitBreaker) Call(ctx context.Context, fn func() error) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	// Check current state
	switch cb.state {
	case StateOpen:
		// Check if we should transition to half-open
		if time.Since(cb.lastFailureTime) > cb.resetTimeout {
			cb.transitionToHalfOpen()
		} else {
			return fmt.Errorf("circuit breaker %s is open", cb.name)
		}

	case StateHalfOpen:
		// Allow limited requests
		if cb.failures+cb.successes >= cb.halfOpenRequests {
			// Evaluate half-open results
			if cb.successes >= cb.halfOpenSuccess {
				cb.transitionToClosed()
			} else {
				cb.transitionToOpen()
				return fmt.Errorf("circuit breaker %s is open", cb.name)
			}
		}
	}

	// Execute the function
	err := fn()

	// Update state based on result
	if err != nil {
		cb.recordFailure()
	} else {
		cb.recordSuccess()
	}

	return err
}

// GetState returns the current state
func (cb *CircuitBreaker) GetState() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Reset manually resets the circuit breaker
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.transitionToClosed()
}

func (cb *CircuitBreaker) recordFailure() {
	cb.failures++
	cb.lastFailureTime = time.Now()

	if cb.state == StateClosed && cb.failures >= cb.maxFailures {
		cb.transitionToOpen()
	}

	cb.logger.Debug("circuit breaker failure recorded",
		slog.String("name", cb.name),
		slog.Int("failures", cb.failures),
		slog.String("state", cb.stateString()))
}

func (cb *CircuitBreaker) recordSuccess() {
	if cb.state == StateHalfOpen {
		cb.successes++
	}

	// Reset failure count on success in closed state
	if cb.state == StateClosed {
		cb.failures = 0
	}

	cb.logger.Debug("circuit breaker success recorded",
		slog.String("name", cb.name),
		slog.Int("successes", cb.successes),
		slog.String("state", cb.stateString()))
}

func (cb *CircuitBreaker) transitionToOpen() {
	cb.state = StateOpen
	cb.logger.Warn("circuit breaker opened",
		slog.String("name", cb.name),
		slog.Int("failures", cb.failures))
}

func (cb *CircuitBreaker) transitionToHalfOpen() {
	cb.state = StateHalfOpen
	cb.failures = 0
	cb.successes = 0
	cb.logger.Info("circuit breaker half-open",
		slog.String("name", cb.name))
}

func (cb *CircuitBreaker) transitionToClosed() {
	cb.state = StateClosed
	cb.failures = 0
	cb.successes = 0
	cb.logger.Info("circuit breaker closed",
		slog.String("name", cb.name))
}

func (cb *CircuitBreaker) stateString() string {
	switch cb.state {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}
