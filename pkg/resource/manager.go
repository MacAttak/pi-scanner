package resource

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/time/rate"
)

// Manager handles resource management including rate limiting,
// connection pooling, and circuit breaking
type Manager struct {
	// Rate limiting
	rateLimiter *rate.Limiter

	// Circuit breakers
	breakers map[string]*CircuitBreaker
	mu       sync.RWMutex

	// Resource tracking
	activeRequests int64
	maxConcurrent  int64
	totalProcessed int64
	totalErrors    int64

	// Connection pool
	semaphore chan struct{}

	logger *slog.Logger
}

// Config holds resource manager configuration
type Config struct {
	// Rate limiting
	RequestsPerSecond float64
	BurstSize         int

	// Concurrency
	MaxConcurrent int

	// Circuit breaker defaults
	MaxFailures  int
	ResetTimeout time.Duration
}

// DefaultConfig returns sensible defaults
func DefaultConfig() Config {
	return Config{
		RequestsPerSecond: 10,
		BurstSize:         20,
		MaxConcurrent:     50,
		MaxFailures:       5,
		ResetTimeout:      30 * time.Second,
	}
}

// NewManager creates a new resource manager
func NewManager(config Config, logger *slog.Logger) *Manager {
	return &Manager{
		rateLimiter:   rate.NewLimiter(rate.Limit(config.RequestsPerSecond), config.BurstSize),
		breakers:      make(map[string]*CircuitBreaker),
		maxConcurrent: int64(config.MaxConcurrent),
		semaphore:     make(chan struct{}, config.MaxConcurrent),
		logger:        logger,
	}
}

// AcquireToken waits for rate limiter permission
func (m *Manager) AcquireToken(ctx context.Context) error {
	return m.rateLimiter.Wait(ctx)
}

// Execute runs a function with resource management
func (m *Manager) Execute(ctx context.Context, resourceName string, fn func() error) error {
	// Rate limiting
	if err := m.AcquireToken(ctx); err != nil {
		return fmt.Errorf("rate limit: %w", err)
	}

	// Get or create circuit breaker
	breaker := m.getOrCreateBreaker(resourceName)

	// Circuit breaker protection
	return breaker.Call(ctx, func() error {
		// Concurrency limiting
		select {
		case m.semaphore <- struct{}{}:
			// Acquired semaphore
			defer func() { <-m.semaphore }()
		case <-ctx.Done():
			return ctx.Err()
		}

		// Track active requests
		atomic.AddInt64(&m.activeRequests, 1)
		defer atomic.AddInt64(&m.activeRequests, -1)

		// Execute function
		err := fn()

		// Update metrics
		atomic.AddInt64(&m.totalProcessed, 1)
		if err != nil {
			atomic.AddInt64(&m.totalErrors, 1)
		}

		return err
	})
}

// GetStats returns current resource statistics
func (m *Manager) GetStats() Stats {
	return Stats{
		ActiveRequests:  atomic.LoadInt64(&m.activeRequests),
		TotalProcessed:  atomic.LoadInt64(&m.totalProcessed),
		TotalErrors:     atomic.LoadInt64(&m.totalErrors),
		RateLimitTokens: m.rateLimiter.Tokens(),
		CircuitBreakers: m.getCircuitBreakerStats(),
	}
}

// Throttle temporarily reduces rate limit
func (m *Manager) Throttle(factor float64) {
	currentLimit := float64(m.rateLimiter.Limit())
	newLimit := currentLimit * factor
	m.rateLimiter.SetLimit(rate.Limit(newLimit))

	m.logger.Info("rate limit throttled",
		slog.Float64("previous", currentLimit),
		slog.Float64("new", newLimit))
}

// RestoreLimit restores original rate limit
func (m *Manager) RestoreLimit(original float64) {
	m.rateLimiter.SetLimit(rate.Limit(original))
	m.logger.Info("rate limit restored",
		slog.Float64("limit", original))
}

// ResetCircuitBreaker manually resets a circuit breaker
func (m *Manager) ResetCircuitBreaker(resourceName string) {
	m.mu.RLock()
	breaker, exists := m.breakers[resourceName]
	m.mu.RUnlock()

	if exists {
		breaker.Reset()
	}
}

func (m *Manager) getOrCreateBreaker(resourceName string) *CircuitBreaker {
	m.mu.RLock()
	breaker, exists := m.breakers[resourceName]
	m.mu.RUnlock()

	if exists {
		return breaker
	}

	// Create new breaker
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if breaker, exists = m.breakers[resourceName]; exists {
		return breaker
	}

	breaker = NewCircuitBreaker(
		resourceName,
		5,              // max failures
		30*time.Second, // reset timeout
		m.logger,
	)
	m.breakers[resourceName] = breaker

	return breaker
}

func (m *Manager) getCircuitBreakerStats() map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]string)
	for name, breaker := range m.breakers {
		stats[name] = breaker.stateString()
	}
	return stats
}

// Stats holds resource manager statistics
type Stats struct {
	ActiveRequests  int64
	TotalProcessed  int64
	TotalErrors     int64
	RateLimitTokens float64
	CircuitBreakers map[string]string
}

// HealthCheck performs a health check on resource manager
func (m *Manager) HealthCheck() error {
	stats := m.GetStats()

	// Check if too many requests are active
	if stats.ActiveRequests > m.maxConcurrent*8/10 {
		return fmt.Errorf("high active requests: %d/%d", stats.ActiveRequests, m.maxConcurrent)
	}

	// Check error rate
	if stats.TotalProcessed > 100 {
		errorRate := float64(stats.TotalErrors) / float64(stats.TotalProcessed)
		if errorRate > 0.1 { // 10% error rate
			return fmt.Errorf("high error rate: %.2f%%", errorRate*100)
		}
	}

	// Check circuit breakers
	openBreakers := 0
	for _, state := range stats.CircuitBreakers {
		if state == "open" {
			openBreakers++
		}
	}

	if openBreakers > len(stats.CircuitBreakers)/2 {
		return fmt.Errorf("too many open circuit breakers: %d", openBreakers)
	}

	return nil
}
