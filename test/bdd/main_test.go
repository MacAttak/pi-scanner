package bdd

import (
	"os"
	"testing"

	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"
)

var opts = godog.Options{
	Output: colors.Colored(os.Stdout),
	Format: "progress", // can be changed to "pretty" for more verbose output
	Paths:  []string{"../../features"},
	Strict: true,
}

func TestFeatures(t *testing.T) {
	// Initialize the test suite
	suite := godog.TestSuite{
		ScenarioInitializer: InitializeScenario,
		Options:             &opts,
	}

	// Run the test suite
	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}

// InitializeScenario registers all step definitions
func InitializeScenario(ctx *godog.ScenarioContext) {
	// CLI step definitions
	registerCLISteps(ctx)

	// Detection step definitions
	registerDetectionSteps(ctx)

	// Common step definitions
	registerCommonSteps(ctx)
}

// Run this to get a list of undefined steps
func TestListUndefinedSteps(t *testing.T) {
	if os.Getenv("GODOG_LIST_UNDEFINED") != "true" {
		t.Skip("Set GODOG_LIST_UNDEFINED=true to list undefined steps")
	}

	opts.Format = "undefined"
	suite := godog.TestSuite{
		ScenarioInitializer: InitializeScenario,
		Options:             &opts,
	}

	suite.Run()
}
