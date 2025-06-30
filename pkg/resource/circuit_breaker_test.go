package resource

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCircuitBreaker(t *testing.T) {
	logger := slog.Default()

	t.Run("starts closed", func(t *testing.T) {
		cb := NewCircuitBreaker("test", 3, time.Second, logger)
		assert.Equal(t, StateClosed, cb.GetState())
	})

	t.Run("opens after max failures", func(t *testing.T) {
		cb := NewCircuitBreaker("test", 3, time.Second, logger)
		ctx := context.Background()

		// Fail 3 times
		for i := 0; i < 3; i++ {
			err := cb.Call(ctx, func() error {
				return errors.New("test error")
			})
			assert.Error(t, err)
		}

		// Should be open now
		assert.Equal(t, StateOpen, cb.GetState())

		// Next call should fail immediately
		err := cb.Call(ctx, func() error {
			return nil
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "circuit breaker test is open")
	})

	t.Run("transitions to half-open after timeout", func(t *testing.T) {
		cb := NewCircuitBreaker("test", 2, 50*time.Millisecond, logger)
		ctx := context.Background()

		// Open the circuit
		for i := 0; i < 2; i++ {
			err := cb.Call(ctx, func() error {
				return errors.New("test error")
			})
			assert.Error(t, err)
		}
		assert.Equal(t, StateOpen, cb.GetState())

		// Wait for reset timeout
		time.Sleep(60 * time.Millisecond)

		// Next call should be allowed (half-open)
		err := cb.Call(ctx, func() error {
			return nil
		})
		assert.NoError(t, err)
	})

	t.Run("closes after successful half-open", func(t *testing.T) {
		cb := NewCircuitBreaker("test", 2, 50*time.Millisecond, logger)
		cb.halfOpenRequests = 3
		cb.halfOpenSuccess = 2
		ctx := context.Background()

		// Open the circuit
		for i := 0; i < 2; i++ {
			err := cb.Call(ctx, func() error {
				return errors.New("test error")
			})
			assert.Error(t, err)
		}

		// Wait for half-open
		time.Sleep(60 * time.Millisecond)

		// Success in half-open
		for i := 0; i < 2; i++ {
			err := cb.Call(ctx, func() error {
				return nil
			})
			assert.NoError(t, err)
		}

		// Need one more call to reach halfOpenRequests threshold
		err := cb.Call(ctx, func() error {
			return nil
		})
		assert.NoError(t, err)

		// The evaluation happens when starting a call that would exceed the threshold
		// So we need one more call to trigger the state transition
		err = cb.Call(ctx, func() error {
			return nil
		})
		assert.NoError(t, err)

		// Should be closed now
		assert.Equal(t, StateClosed, cb.GetState())
	})

	t.Run("returns to open on half-open failures", func(t *testing.T) {
		cb := NewCircuitBreaker("test", 2, 50*time.Millisecond, logger)
		cb.halfOpenRequests = 3
		cb.halfOpenSuccess = 2
		ctx := context.Background()

		// Open the circuit
		for i := 0; i < 2; i++ {
			err := cb.Call(ctx, func() error {
				return errors.New("test error")
			})
			assert.Error(t, err)
		}

		// Wait for half-open
		time.Sleep(60 * time.Millisecond)

		// Fail in half-open
		for i := 0; i < 2; i++ {
			err := cb.Call(ctx, func() error {
				return errors.New("still failing")
			})
			assert.Error(t, err)
		}

		// Need one more call to reach halfOpenRequests threshold
		err := cb.Call(ctx, func() error {
			return errors.New("still failing")
		})
		assert.Error(t, err)

		// The evaluation happens when starting a call that would exceed the threshold
		// So we need one more call to trigger the state transition
		err = cb.Call(ctx, func() error {
			return errors.New("still failing")
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "circuit breaker test is open")

		// Should be open again
		assert.Equal(t, StateOpen, cb.GetState())
	})

	t.Run("manual reset", func(t *testing.T) {
		cb := NewCircuitBreaker("test", 1, time.Hour, logger)
		ctx := context.Background()

		// Open the circuit
		err := cb.Call(ctx, func() error {
			return errors.New("test error")
		})
		assert.Error(t, err)
		assert.Equal(t, StateOpen, cb.GetState())

		// Manual reset
		cb.Reset()
		assert.Equal(t, StateClosed, cb.GetState())

		// Should work now
		err = cb.Call(ctx, func() error {
			return nil
		})
		assert.NoError(t, err)
	})
}

func TestCircuitBreakerConcurrency(t *testing.T) {
	logger := slog.Default()
	cb := NewCircuitBreaker("test", 10, time.Second, logger)
	ctx := context.Background()

	// Run concurrent operations
	done := make(chan bool, 100)
	for i := 0; i < 100; i++ {
		go func(i int) {
			defer func() { done <- true }()

			err := cb.Call(ctx, func() error {
				if i%3 == 0 {
					return errors.New("test error")
				}
				return nil
			})

			// Don't assert on error as circuit may open
			_ = err
		}(i)
	}

	// Wait for all to complete
	for i := 0; i < 100; i++ {
		<-done
	}

	// Should have handled all requests without panic
	state := cb.GetState()
	require.Contains(t, []CircuitState{StateClosed, StateOpen}, state)
}
