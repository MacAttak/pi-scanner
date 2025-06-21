package test

import (
	"os"
	"strconv"
	"time"
)

// TestConfig provides configuration for E2E tests
type TestConfig struct {
	SkipNetworkTests bool
	SkipAuthTests    bool
	UseShallowClone  bool
	DefaultTimeout   time.Duration
	MaxRetries       int
	RetryDelay       time.Duration
	VerboseOutput    bool
	TestDataDir      string
	EnableMockRepos  bool
	GitHubToken      string
	TestRepositories []TestRepositoryConfig
}

// TestRepositoryConfig configures individual test repositories
type TestRepositoryConfig struct {
	Name                string
	URL                 string
	Description         string
	ExpectedMinFiles    int
	ExpectedMaxDuration time.Duration
	ExpectedPITypes     []string
	ExpectedMinFindings int
	ExpectedMaxFindings int
	SkipIfNoAuth        bool
	RequireAuth         bool
}

// GetTestConfig returns the test configuration based on environment variables
func GetTestConfig() *TestConfig {
	config := &TestConfig{
		SkipNetworkTests: getBoolEnv("SKIP_NETWORK_TESTS", false),
		SkipAuthTests:    getBoolEnv("SKIP_AUTH_TESTS", false),
		UseShallowClone:  getBoolEnv("USE_SHALLOW_CLONE", true),
		DefaultTimeout:   getDurationEnv("DEFAULT_TIMEOUT", 60*time.Second),
		MaxRetries:       getIntEnv("MAX_RETRIES", 3),
		RetryDelay:       getDurationEnv("RETRY_DELAY", 2*time.Second),
		VerboseOutput:    getBoolEnv("VERBOSE_OUTPUT", false),
		TestDataDir:      getStringEnv("TEST_DATA_DIR", "testdata"),
		EnableMockRepos:  getBoolEnv("ENABLE_MOCK_REPOS", true),
		GitHubToken:      getStringEnv("GITHUB_TOKEN", ""),
		TestRepositories: getDefaultTestRepositories(),
	}

	// Auto-detect CI environment
	if isCI() {
		config.SkipNetworkTests = getBoolEnv("SKIP_NETWORK_TESTS", true)
		config.UseShallowClone = true
		config.DefaultTimeout = 30 * time.Second
		config.MaxRetries = 2
	}

	// Auto-detect Docker environment
	if isDocker() {
		config.SkipAuthTests = getBoolEnv("SKIP_AUTH_TESTS", true)
		config.EnableMockRepos = true
	}

	return config
}

// ShouldSkipTest determines if a test should be skipped based on configuration
func (c *TestConfig) ShouldSkipTest(testType string, requiresAuth bool, requiresNetwork bool) (bool, string) {
	if requiresNetwork && c.SkipNetworkTests {
		return true, "network tests disabled"
	}

	if requiresAuth && c.SkipAuthTests {
		return true, "auth tests disabled"
	}

	if requiresAuth && c.GitHubToken == "" && !isCI() {
		return true, "no GitHub token available"
	}

	return false, ""
}

// GetTestRepository returns a test repository by name
func (c *TestConfig) GetTestRepository(name string) *TestRepositoryConfig {
	for _, repo := range c.TestRepositories {
		if repo.Name == name {
			return &repo
		}
	}
	return nil
}

// Helper functions

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if result, err := strconv.ParseBool(value); err == nil {
			return result
		}
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if result, err := strconv.Atoi(value); err == nil {
			return result
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if result, err := time.ParseDuration(value); err == nil {
			return result
		}
	}
	return defaultValue
}

func getStringEnv(key string, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func isCI() bool {
	return os.Getenv("CI") != "" ||
		os.Getenv("GITHUB_ACTIONS") != "" ||
		os.Getenv("JENKINS_URL") != "" ||
		os.Getenv("BUILDKITE") != ""
}

func isDocker() bool {
	// Check for Docker-specific environment
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}

	// Check for common Docker environment variables
	return os.Getenv("DOCKER_CONTAINER") != "" ||
		os.Getenv("CONTAINER") != ""
}

// getDefaultTestRepositories returns the default set of test repositories
func getDefaultTestRepositories() []TestRepositoryConfig {
	return []TestRepositoryConfig{
		{
			Name:                "Small Public Repository",
			URL:                 "https://github.com/octocat/Hello-World",
			Description:         "Small test repository for basic functionality",
			ExpectedMinFiles:    1,
			ExpectedMaxDuration: 30 * time.Second,
			ExpectedPITypes:     []string{},
			ExpectedMinFindings: 0,
			ExpectedMaxFindings: 10,
			SkipIfNoAuth:        false,
			RequireAuth:         false,
		},
		{
			Name:                "GitHub Documentation",
			URL:                 "https://github.com/github/docs",
			Description:         "GitHub's documentation repository",
			ExpectedMinFiles:    1000,
			ExpectedMaxDuration: 120 * time.Second,
			ExpectedPITypes:     []string{"EMAIL", "NAME"},
			ExpectedMinFindings: 50,
			ExpectedMaxFindings: 5000,
			SkipIfNoAuth:        true,
			RequireAuth:         true,
		},
		{
			Name:                "FreeCodeCamp",
			URL:                 "https://github.com/freeCodeCamp/freeCodeCamp",
			Description:         "Educational platform with diverse content",
			ExpectedMinFiles:    5000,
			ExpectedMaxDuration: 180 * time.Second,
			ExpectedPITypes:     []string{"NAME", "EMAIL"},
			ExpectedMinFindings: 100,
			ExpectedMaxFindings: 10000,
			SkipIfNoAuth:        true,
			RequireAuth:         true,
		},
	}
}
