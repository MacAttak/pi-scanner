package config_test

import (
	"fmt"
	"log"
	"os"

	"github.com/MacAttak/pi-scanner/pkg/config"
)

func ExampleLoadConfig() {
	// Load configuration from file
	cfg, err := config.LoadConfig("scanner.yaml")
	if err != nil {
		// Fall back to defaults if config file not found
		cfg = config.DefaultConfig()
	}

	fmt.Printf("Workers: %d\n", cfg.Scanner.Workers)
	fmt.Printf("Risk Threshold (Critical): %.1f\n", cfg.Risk.Thresholds.Critical)

	// Output:
	// Workers: 4
	// Risk Threshold (Critical): 0.8
}

func ExampleGenerateExampleConfig() {
	// Generate an example configuration file
	if err := config.GenerateExampleConfig("example-config.yaml"); err != nil {
		log.Fatal(err)
	}

	// Load and display the generated config
	cfg, err := config.LoadConfig("example-config.yaml")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Generated config version: %s\n", cfg.Version)
	fmt.Printf("Scanner workers: %d\n", cfg.Scanner.Workers)

	// Clean up
	os.Remove("example-config.yaml")

	// Output:
	// Generated config version: 1.0
	// Scanner workers: 8
}

func ExampleConfig_Validate() {
	cfg := config.DefaultConfig()

	// Modify some values
	cfg.Scanner.Workers = 16
	cfg.Risk.Thresholds.Critical = 0.9

	// Validate the configuration
	if err := cfg.Validate(); err != nil {
		fmt.Printf("Configuration error: %v\n", err)
	} else {
		fmt.Println("Configuration is valid")
	}

	// Output:
	// Configuration is valid
}

func ExampleMergeConfig() {
	// Start with default configuration
	base := config.DefaultConfig()

	// Create override configuration
	override := &config.Config{
		Scanner: config.ScannerConfig{
			Workers: 8,
		},
	}

	// Merge configurations
	merged := config.MergeConfig(base, override)

	fmt.Printf("Workers: %d\n", merged.Scanner.Workers)
	fmt.Printf("File Types Count: %d\n", len(merged.Scanner.FileTypes))

	// Output:
	// Workers: 8
	// File Types Count: 34
}