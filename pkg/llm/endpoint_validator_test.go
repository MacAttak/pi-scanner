package llm

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEndpointValidator_ValidateEndpoint(t *testing.T) {
	tests := []struct {
		name          string
		config        ValidationConfig
		endpoint      string
		expectError   bool
		errorContains string
	}{
		{
			name:        "localhost http allowed in strict mode",
			config:      DefaultValidationConfig(),
			endpoint:    "http://localhost:1234/v1",
			expectError: false,
		},
		{
			name:        "127.0.0.1 http allowed in strict mode",
			config:      DefaultValidationConfig(),
			endpoint:    "http://127.0.0.1:1234/v1",
			expectError: false,
		},
		{
			name:        "::1 http allowed in strict mode",
			config:      DefaultValidationConfig(),
			endpoint:    "http://[::1]:1234/v1",
			expectError: false,
		},
		{
			name:          "external domain blocked in strict mode",
			config:        DefaultValidationConfig(),
			endpoint:      "http://api.openai.com/v1",
			expectError:   true,
			errorContains: "strict mode enabled",
		},
		{
			name:          "private IP blocked in strict mode",
			config:        DefaultValidationConfig(),
			endpoint:      "http://192.168.1.100:1234/v1",
			expectError:   true,
			errorContains: "strict mode enabled",
		},
		{
			name: "private IP allowed when configured",
			config: ValidationConfig{
				AllowedHosts:     []string{"localhost"},
				AllowedNetworks:  []string{"192.168.0.0/16"},
				AllowPrivateIPs:  true,
				EnableStrictMode: false,
			},
			endpoint:    "http://192.168.1.100:1234/v1",
			expectError: false,
		},
		{
			name: "https required for non-localhost",
			config: ValidationConfig{
				AllowedHosts:     []string{"localhost", "myserver.local"},
				RequireHTTPS:     true,
				EnableStrictMode: false,
			},
			endpoint:      "http://myserver.local:1234/v1",
			expectError:   true,
			errorContains: "HTTPS required",
		},
		{
			name: "https not required for localhost",
			config: ValidationConfig{
				AllowedHosts:     []string{"localhost"},
				RequireHTTPS:     true,
				EnableStrictMode: false,
			},
			endpoint:    "http://localhost:1234/v1",
			expectError: false,
		},
		{
			name:          "missing scheme",
			config:        DefaultValidationConfig(),
			endpoint:      "localhost:1234/v1",
			expectError:   true,
			errorContains: "unsupported scheme", // URL parser interprets "localhost" as the scheme
		},
		{
			name:          "invalid scheme",
			config:        DefaultValidationConfig(),
			endpoint:      "ftp://localhost:1234/v1",
			expectError:   true,
			errorContains: "unsupported scheme",
		},
		{
			name:          "missing hostname",
			config:        DefaultValidationConfig(),
			endpoint:      "http:///v1",
			expectError:   true,
			errorContains: "must include hostname",
		},
		{
			name: "allowed specific host",
			config: ValidationConfig{
				AllowedHosts:     []string{"llm.internal"},
				EnableStrictMode: false,
			},
			endpoint:    "http://llm.internal:8080/v1",
			expectError: false,
		},
		{
			name: "IPv6 loopback allowed",
			config: ValidationConfig{
				AllowedNetworks:  []string{"::1/128"},
				EnableStrictMode: false,
			},
			endpoint:    "http://[::1]:1234/v1",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewEndpointValidator(tt.config)
			err := validator.ValidateEndpoint(tt.endpoint)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEndpointValidator_ValidateConfig(t *testing.T) {
	validator := NewEndpointValidator(DefaultValidationConfig())

	tests := []struct {
		name          string
		config        Config
		expectError   bool
		errorContains string
	}{
		{
			name: "valid config",
			config: Config{
				Endpoint:    "http://localhost:1234/v1",
				MaxTokens:   1000,
				Temperature: 0.7,
				Timeout:     30 * time.Second,
			},
			expectError: false,
		},
		{
			name: "invalid endpoint",
			config: Config{
				Endpoint:    "http://api.openai.com/v1",
				MaxTokens:   1000,
				Temperature: 0.7,
				Timeout:     30 * time.Second,
			},
			expectError:   true,
			errorContains: "endpoint validation failed",
		},
		{
			name: "max tokens too low",
			config: Config{
				Endpoint:    "http://localhost:1234/v1",
				MaxTokens:   50,
				Temperature: 0.7,
				Timeout:     30 * time.Second,
			},
			expectError:   true,
			errorContains: "must be at least 100",
		},
		{
			name: "max tokens too high",
			config: Config{
				Endpoint:    "http://localhost:1234/v1",
				MaxTokens:   5000,
				Temperature: 0.7,
				Timeout:     30 * time.Second,
			},
			expectError:   true,
			errorContains: "exceeds maximum of 4096",
		},
		{
			name: "temperature too low",
			config: Config{
				Endpoint:    "http://localhost:1234/v1",
				MaxTokens:   1000,
				Temperature: -0.1,
				Timeout:     30 * time.Second,
			},
			expectError:   true,
			errorContains: "temperature must be between",
		},
		{
			name: "temperature too high",
			config: Config{
				Endpoint:    "http://localhost:1234/v1",
				MaxTokens:   1000,
				Temperature: 2.1,
				Timeout:     30 * time.Second,
			},
			expectError:   true,
			errorContains: "temperature must be between",
		},
		{
			name: "timeout too short",
			config: Config{
				Endpoint:    "http://localhost:1234/v1",
				MaxTokens:   1000,
				Temperature: 0.7,
				Timeout:     500 * time.Millisecond,
			},
			expectError:   true,
			errorContains: "must be at least 1 second",
		},
		{
			name: "timeout too long",
			config: Config{
				Endpoint:    "http://localhost:1234/v1",
				MaxTokens:   1000,
				Temperature: 0.7,
				Timeout:     10 * time.Minute,
			},
			expectError:   true,
			errorContains: "exceeds maximum of 5 minutes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateConfig(tt.config)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEndpointValidator_ValidateProviderConfig(t *testing.T) {
	validator := NewEndpointValidator(DefaultValidationConfig())

	tests := []struct {
		name          string
		provider      string
		config        Config
		expectError   bool
		errorContains string
	}{
		{
			name:     "lmstudio localhost allowed",
			provider: "lmstudio",
			config: Config{
				Endpoint: "http://localhost:1234/v1",
			},
			expectError: false,
		},
		{
			name:     "lmstudio external blocked",
			provider: "lmstudio",
			config: Config{
				Endpoint: "http://external.com:1234/v1",
			},
			expectError:   true,
			errorContains: "must run on localhost",
		},
		{
			name:     "ollama localhost allowed",
			provider: "ollama",
			config: Config{
				Endpoint: "http://localhost:11434/v1",
			},
			expectError: false,
		},
		{
			name:     "openai blocked",
			provider: "openai",
			config: Config{
				Endpoint: "https://api.openai.com/v1",
			},
			expectError:   true,
			errorContains: "not allowed for PI data",
		},
		{
			name:     "anthropic blocked",
			provider: "anthropic",
			config: Config{
				Endpoint: "https://api.anthropic.com/v1",
			},
			expectError:   true,
			errorContains: "not allowed for PI data",
		},
		{
			name:     "unknown provider must use localhost",
			provider: "custom-llm",
			config: Config{
				Endpoint: "http://localhost:8080/v1",
			},
			expectError: false,
		},
		{
			name:     "unknown provider external blocked",
			provider: "custom-llm",
			config: Config{
				Endpoint: "http://external.llm.com/v1",
			},
			expectError:   true,
			errorContains: "must use localhost endpoint",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateProviderConfig(tt.provider, tt.config)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEndpointValidator_isLocalhost(t *testing.T) {
	validator := NewEndpointValidator(DefaultValidationConfig())

	tests := []struct {
		hostname string
		expected bool
	}{
		{"localhost", true},
		{"LOCALHOST", true},
		{"127.0.0.1", true},
		{"::1", true},
		{"example.com", false},
		{"192.168.1.1", false},
		{"10.0.0.1", false},
		{"localhost.com", false},
		{"127.0.0.2", true}, // Still in loopback range
	}

	for _, tt := range tests {
		t.Run(tt.hostname, func(t *testing.T) {
			result := validator.isLocalhost(tt.hostname)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEndpointValidator_isPrivateIP(t *testing.T) {
	validator := NewEndpointValidator(DefaultValidationConfig())

	tests := []struct {
		ip       string
		expected bool
	}{
		{"10.0.0.1", true},
		{"10.255.255.254", true},
		{"172.16.0.1", true},
		{"172.31.255.254", true},
		{"192.168.0.1", true},
		{"192.168.255.254", true},
		{"8.8.8.8", false},
		{"1.1.1.1", false},
		{"127.0.0.1", false},   // Loopback, not private
		{"::1", false},         // IPv6 loopback
		{"fc00::1", true},      // IPv6 private
		{"2001:db8::1", false}, // IPv6 public
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			require.NotNil(t, ip, "Failed to parse IP: %s", tt.ip)

			result := validator.isPrivateIP(ip)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEndpointValidator_TestEndpoint(t *testing.T) {
	// Note: This test would require a mock LLM server or skip if no server is available
	t.Skip("Requires running LLM server")

	validator := NewEndpointValidator(DefaultValidationConfig())

	ctx := context.Background()
	err := validator.TestEndpoint(ctx, "http://localhost:1234/v1")

	// The test would fail if no LM Studio is running
	// In a real test environment, we'd mock this
	assert.Error(t, err)
}

func TestNewEndpointValidator_Defaults(t *testing.T) {
	// Test with empty config
	validator := NewEndpointValidator(ValidationConfig{})

	// Should use defaults
	err := validator.ValidateEndpoint("http://localhost:1234/v1")
	assert.NoError(t, err)

	// Should block external in strict mode
	err = validator.ValidateEndpoint("http://example.com:1234/v1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "strict mode")
}

func TestEndpointValidator_ComplexScenarios(t *testing.T) {
	t.Run("multiple allowed networks", func(t *testing.T) {
		config := ValidationConfig{
			AllowedNetworks: []string{
				"10.0.0.0/24",
				"192.168.1.0/24",
				"172.16.0.0/16",
			},
			AllowPrivateIPs:  true,
			EnableStrictMode: false,
		}
		validator := NewEndpointValidator(config)

		// These should be allowed
		endpoints := []string{
			"http://10.0.0.5:8080/v1",
			"http://192.168.1.100:8080/v1",
			"http://172.16.5.10:8080/v1",
		}

		for _, endpoint := range endpoints {
			err := validator.ValidateEndpoint(endpoint)
			assert.NoError(t, err, "Endpoint should be allowed: %s", endpoint)
		}

		// This should be blocked (different subnet)
		err := validator.ValidateEndpoint("http://10.0.1.5:8080/v1")
		assert.Error(t, err)
	})

	t.Run("hostname resolution", func(t *testing.T) {
		// This test depends on DNS resolution
		config := ValidationConfig{
			AllowedHosts:     []string{"localhost"},
			EnableStrictMode: false,
		}
		validator := NewEndpointValidator(config)

		// localhost should resolve and be allowed
		err := validator.ValidateEndpoint("http://localhost:8080/v1")
		assert.NoError(t, err)
	})
}
