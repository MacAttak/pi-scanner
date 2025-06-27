package llm

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"
)

// EndpointValidator validates LLM endpoints for security compliance
type EndpointValidator struct {
	config ValidationConfig
}

// ValidationConfig configures endpoint validation
type ValidationConfig struct {
	// AllowedHosts is a list of allowed hostnames (default: localhost only)
	AllowedHosts []string

	// AllowedNetworks is a list of allowed IP networks in CIDR notation
	AllowedNetworks []string

	// AllowPrivateIPs allows RFC1918 private IP addresses
	AllowPrivateIPs bool

	// RequireHTTPS enforces HTTPS for non-localhost endpoints
	RequireHTTPS bool

	// MaxResponseTime is the maximum allowed response time for health checks
	MaxResponseTime time.Duration

	// EnableStrictMode enforces localhost-only connections
	EnableStrictMode bool
}

// DefaultValidationConfig returns a secure default configuration
func DefaultValidationConfig() ValidationConfig {
	return ValidationConfig{
		AllowedHosts:     []string{"localhost", "127.0.0.1", "::1"},
		AllowedNetworks:  []string{"127.0.0.0/8", "::1/128"},
		AllowPrivateIPs:  false,
		RequireHTTPS:     false, // Not required for localhost
		MaxResponseTime:  5 * time.Second,
		EnableStrictMode: true, // Enforce localhost-only by default
	}
}

// NewEndpointValidator creates a new endpoint validator
func NewEndpointValidator(config ValidationConfig) *EndpointValidator {
	// Set defaults if not specified
	if len(config.AllowedHosts) == 0 && len(config.AllowedNetworks) == 0 {
		config = DefaultValidationConfig()
	}
	if config.MaxResponseTime == 0 {
		config.MaxResponseTime = 5 * time.Second
	}

	return &EndpointValidator{
		config: config,
	}
}

// ValidateEndpoint validates an LLM endpoint URL
func (v *EndpointValidator) ValidateEndpoint(endpoint string) error {
	// Parse the URL
	u, err := url.Parse(endpoint)
	if err != nil {
		return fmt.Errorf("invalid endpoint URL: %w", err)
	}

	// Check scheme
	if u.Scheme == "" {
		return errors.New("endpoint URL must include scheme (http:// or https://)")
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("unsupported scheme: %s", u.Scheme)
	}

	// Extract hostname and port
	hostname := u.Hostname()
	if hostname == "" {
		return errors.New("endpoint URL must include hostname")
	}

	// In strict mode, only allow localhost
	if v.config.EnableStrictMode {
		if !v.isLocalhost(hostname) {
			return fmt.Errorf("strict mode enabled: only localhost endpoints are allowed, got %s", hostname)
		}
		return nil
	}

	// Check if hostname is in allowed list
	if v.isHostnameAllowed(hostname) {
		// Still need to check HTTPS requirement
		if v.config.RequireHTTPS && !v.isLocalhost(hostname) && u.Scheme != "https" {
			return fmt.Errorf("HTTPS required for non-localhost endpoint: %s", endpoint)
		}
		return nil
	}

	// Try to resolve hostname to IP
	ips, err := net.LookupIP(hostname)
	if err != nil {
		// If we can't resolve the hostname, check if HTTPS is required
		if v.config.RequireHTTPS && !v.isLocalhost(hostname) && u.Scheme != "https" {
			return fmt.Errorf("HTTPS required for non-localhost endpoint: %s", endpoint)
		}
		return fmt.Errorf("failed to resolve hostname %s: %w", hostname, err)
	}

	// Check each resolved IP
	validIPFound := false
	for _, ip := range ips {
		if err := v.validateIP(ip); err == nil {
			// At least one IP is valid
			validIPFound = true
			break
		}
	}

	if !validIPFound {
		return fmt.Errorf("endpoint %s resolves to disallowed IP addresses", hostname)
	}

	// Check HTTPS requirement for non-localhost
	if v.config.RequireHTTPS && !v.isLocalhost(hostname) && u.Scheme != "https" {
		return fmt.Errorf("HTTPS required for non-localhost endpoint: %s", endpoint)
	}

	return nil
}

// ValidateConfig validates the LLM configuration
func (v *EndpointValidator) ValidateConfig(config Config) error {
	// Validate endpoint
	if err := v.ValidateEndpoint(config.Endpoint); err != nil {
		return fmt.Errorf("endpoint validation failed: %w", err)
	}

	// Validate other settings
	if config.MaxTokens < 100 {
		return errors.New("max_tokens must be at least 100")
	}

	if config.MaxTokens > 4096 {
		return errors.New("max_tokens exceeds maximum of 4096")
	}

	if config.Temperature < 0 || config.Temperature > 2 {
		return errors.New("temperature must be between 0 and 2")
	}

	if config.Timeout < 1*time.Second {
		return errors.New("timeout must be at least 1 second")
	}

	if config.Timeout > 5*time.Minute {
		return errors.New("timeout exceeds maximum of 5 minutes")
	}

	return nil
}

// TestEndpoint performs a health check on the endpoint
func (v *EndpointValidator) TestEndpoint(ctx context.Context, endpoint string) error {
	// First validate the endpoint
	if err := v.ValidateEndpoint(endpoint); err != nil {
		return err
	}

	// Create a client for testing
	testConfig := Config{
		Endpoint: endpoint,
		APIKey:   "test",
		Model:    "test",
		Timeout:  v.config.MaxResponseTime,
	}

	client, err := NewLMStudioClient(testConfig)
	if err != nil {
		return fmt.Errorf("failed to create test client: %w", err)
	}

	// Set timeout for health check
	ctx, cancel := context.WithTimeout(ctx, v.config.MaxResponseTime)
	defer cancel()

	// Perform health check
	if err := client.HealthCheck(ctx); err != nil {
		return fmt.Errorf("endpoint health check failed: %w", err)
	}

	return nil
}

// isLocalhost checks if a hostname is localhost
func (v *EndpointValidator) isLocalhost(hostname string) bool {
	localhostNames := []string{"localhost", "127.0.0.1", "::1"}
	hostname = strings.ToLower(hostname)

	for _, name := range localhostNames {
		if hostname == name {
			return true
		}
	}

	// Check if it's a loopback IP
	if ip := net.ParseIP(hostname); ip != nil {
		return ip.IsLoopback()
	}

	return false
}

// isHostnameAllowed checks if a hostname is in the allowed list
func (v *EndpointValidator) isHostnameAllowed(hostname string) bool {
	hostname = strings.ToLower(hostname)

	for _, allowed := range v.config.AllowedHosts {
		if strings.ToLower(allowed) == hostname {
			return true
		}
	}

	return false
}

// validateIP validates an IP address against allowed networks
func (v *EndpointValidator) validateIP(ip net.IP) error {
	// Check if it's a loopback address
	if ip.IsLoopback() {
		return nil
	}

	// Check private IPs
	if v.isPrivateIP(ip) {
		if !v.config.AllowPrivateIPs {
			return fmt.Errorf("private IP addresses not allowed: %s", ip)
		}
		// If private IPs are allowed, still need to check against allowed networks
	}

	// Check against allowed networks
	for _, network := range v.config.AllowedNetworks {
		_, ipNet, err := net.ParseCIDR(network)
		if err != nil {
			continue // Skip invalid CIDR
		}

		if ipNet.Contains(ip) {
			return nil
		}
	}

	return fmt.Errorf("IP address not in allowed networks: %s", ip)
}

// isPrivateIP checks if an IP is in private address space (RFC1918)
func (v *EndpointValidator) isPrivateIP(ip net.IP) bool {
	privateNetworks := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"fc00::/7", // IPv6 private
	}

	for _, network := range privateNetworks {
		_, ipNet, err := net.ParseCIDR(network)
		if err != nil {
			continue
		}

		if ipNet.Contains(ip) {
			return true
		}
	}

	return false
}

// ValidateProviderConfig validates provider-specific configurations
func (v *EndpointValidator) ValidateProviderConfig(provider string, config Config) error {
	switch strings.ToLower(provider) {
	case "lmstudio", "lm-studio":
		// LM Studio specific validation
		u, err := url.Parse(config.Endpoint)
		if err != nil {
			return fmt.Errorf("invalid endpoint URL: %w", err)
		}
		if !v.isLocalhost(u.Hostname()) {
			return errors.New("LM Studio must run on localhost")
		}

	case "ollama":
		// Ollama specific validation
		u, err := url.Parse(config.Endpoint)
		if err != nil {
			return fmt.Errorf("invalid endpoint URL: %w", err)
		}
		if !v.isLocalhost(u.Hostname()) {
			return errors.New("Ollama must run on localhost")
		}

	case "openai":
		// OpenAI should never be used for PI data
		return errors.New("OpenAI API not allowed for PI data - use local LLM only")

	case "anthropic", "claude":
		// Anthropic should never be used for PI data
		return errors.New("Anthropic API not allowed for PI data - use local LLM only")

	default:
		// Unknown provider - apply strict validation
		u, err := url.Parse(config.Endpoint)
		if err != nil {
			return fmt.Errorf("invalid endpoint URL: %w", err)
		}
		if !v.isLocalhost(u.Hostname()) {
			return fmt.Errorf("unknown provider '%s' must use localhost endpoint", provider)
		}
	}

	return nil
}
