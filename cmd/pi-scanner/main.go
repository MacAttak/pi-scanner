package main

import (
	"bufio"
	"context"
	"fmt"
	"net/url"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/MacAttak/pi-scanner/pkg/llm"
	"github.com/spf13/cobra"
)

var (
	// Version information (set by build flags)
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

// Global flags
var (
	noInput      bool
	validateMode string // none, high, high-medium, all
	maskingLevel string
	verbose      bool
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "pi-scanner [repository-url]",
		Short: "Detect personally identifiable information in GitHub repositories",
		Long: `🔒 PI Scanner - Australian Privacy Compliance

Detects personally identifiable information in code repositories
with a focus on Australian regulatory compliance.

This tool provides a guided experience through:
  1. Pattern-based scanning for PI detection
  2. Optional AI-powered validation to reduce false positives

Examples:
  # Interactive guided scan
  pi-scanner https://github.com/example/repo

  # Non-interactive scan (pattern only)
  pi-scanner https://github.com/example/repo --no-input

  # Automated scan with high-risk validation
  pi-scanner https://github.com/example/repo --no-input --validate=high`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// If no args, show help
			if len(args) == 0 {
				return cmd.Help()
			}

			repoURL := args[0]

			// Validate repository URL
			if err := validateRepositoryURL(repoURL); err != nil {
				return fmt.Errorf("invalid repository URL: %w", err)
			}

			// Run the guided scan
			return runGuidedScan(cmd.Context(), repoURL)
		},
	}

	// Global flags
	rootCmd.PersistentFlags().BoolVar(&noInput, "no-input", false, "Disable all interactive prompts")
	rootCmd.PersistentFlags().StringVar(&validateMode, "validate", "", "Validation mode: none, high, high-medium, all (requires --no-input)")
	rootCmd.PersistentFlags().StringVar(&maskingLevel, "masking", "partial", "Masking level for PI data: none, partial, full")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")

	// Add subcommands
	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newLLMCheckCmd())

	return rootCmd
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Display version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(cmd.OutOrStdout(), "PI Scanner\n")
			fmt.Fprintf(cmd.OutOrStdout(), "Version: %s\n", version)
			fmt.Fprintf(cmd.OutOrStdout(), "Build: %s\n", commit)
			fmt.Fprintf(cmd.OutOrStdout(), "Build Date: %s\n", buildDate)
			fmt.Fprintf(cmd.OutOrStdout(), "Go Version: %s\n", runtime.Version())
			fmt.Fprintf(cmd.OutOrStdout(), "OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
		},
	}
}

func newLLMCheckCmd() *cobra.Command {
	var (
		endpoint string
		model    string
	)

	cmd := &cobra.Command{
		Use:   "llm-check",
		Short: "Check LLM service status and configuration",
		Long: `Check if the LLM service is available and properly configured.
This command helps diagnose LLM connectivity issues.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Printf("🤖 LLM Service Check\n")
			cmd.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

			cmd.Printf("Endpoint: %s\n", endpoint)
			cmd.Printf("Model: %s\n\n", model)

			// Create LLM client
			config := llm.Config{
				Enabled:     true,
				Provider:    "lmstudio",
				Endpoint:    endpoint,
				Model:       model,
				APIKey:      "lm-studio",
				MaxTokens:   1000,
				Temperature: 0.3,
				Timeout:     5 * time.Second,
			}

			client, err := llm.NewLMStudioClient(config)
			if err != nil {
				cmd.Printf("❌ Failed to create LLM client: %v\n", err)
				return err
			}

			// Check health
			cmd.Printf("🔍 Checking service availability...\n")
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			err = client.HealthCheck(ctx)
			if err != nil {
				cmd.Printf("\n❌ LLM Service Unavailable\n\n")
				cmd.Printf("Error: %v\n\n", err)
				cmd.Printf("Troubleshooting steps:\n\n")
				cmd.Printf("1. Ensure LM Studio is installed: https://lmstudio.ai/\n")
				cmd.Printf("2. Start LM Studio and load a model\n")
				cmd.Printf("3. Start the local server (usually on port 1234)\n")
				cmd.Printf("4. Check the server is running at: %s\n", endpoint)
				cmd.Printf("5. Try accessing %s/v1/models in your browser\n\n", strings.TrimSuffix(endpoint, "/v1"))

				if strings.Contains(err.Error(), "connection refused") {
					cmd.Printf("💡 The server appears to be not running.\n")
					cmd.Printf("   Start the server in LM Studio first.\n")
				} else if strings.Contains(err.Error(), "timeout") {
					cmd.Printf("💡 The server is not responding.\n")
					cmd.Printf("   Check if the endpoint URL is correct.\n")
				}

				return fmt.Errorf("LLM service check failed")
			}

			cmd.Printf("\n✅ LLM Service Available!\n\n")
			cmd.Printf("The LLM service is running and ready for use.\n")

			if verbose {
				cmd.Printf("\nService Details:\n")
				cmd.Printf("- Provider: LM Studio\n")
				cmd.Printf("- API: OpenAI Compatible\n")
				cmd.Printf("- Endpoint: %s\n", endpoint)
				cmd.Printf("- Model: %s\n", model)
			}

			return nil
		},
	}

	// Add flags
	cmd.Flags().StringVar(&endpoint, "endpoint", "http://localhost:1234/v1", "LLM API endpoint")
	cmd.Flags().StringVar(&model, "model", "qwen2.5-coder-7b-instruct", "LLM model name")

	return cmd
}

// validateRepositoryURL validates that the provided URL is a valid repository URL
func validateRepositoryURL(repoURL string) error {
	// Parse the URL
	u, err := url.Parse(repoURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %v", err)
	}

	// Must have a scheme
	if u.Scheme == "" {
		return fmt.Errorf("URL must include protocol (http:// or https://)")
	}

	// Must have a host
	if u.Host == "" {
		return fmt.Errorf("URL must include a host")
	}

	// Check if it looks like a GitHub URL
	if strings.Contains(u.Host, "github.com") {
		// Basic GitHub URL validation
		pathParts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(pathParts) < 2 {
			return fmt.Errorf("GitHub URL must be in format: https://github.com/owner/repo")
		}
	}

	return nil
}

// runGuidedScan runs the complete guided scanning experience
func runGuidedScan(ctx context.Context, repoURL string) error {
	// Show welcome screen
	if !noInput {
		displayWelcome()
	}

	// Create resource manager for entire scan lifecycle
	fmt.Printf("🔐 Checking GitHub authentication...")
	resourceManager, err := NewScanResourceManager(ctx, repoURL)
	if err != nil {
		fmt.Printf(" ❌\n")
		return fmt.Errorf("failed to initialize scan resources: %w", err)
	}
	defer func() {
		if err := resourceManager.Cleanup(); err != nil {
			fmt.Printf("⚠️  Warning: Failed to cleanup resources: %v\n", err)
		}
	}()
	fmt.Printf(" ✅\n")

	fmt.Printf("📥 Cloning repository... ✅\n")

	// Phase 1: Pattern-based scanning
	fmt.Printf("\n🔍 Phase 1: Pattern-based scanning\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	scanResult, reportDir, err := runPatternScan(ctx, resourceManager, repoURL)
	if err != nil {
		return fmt.Errorf("pattern scan failed: %w", err)
	}

	// Display results
	displayScanSummary(scanResult)
	fmt.Printf("\n📁 Pattern scan report saved to: %s\n", reportDir)

	// Check if we should run LLM validation
	shouldValidate := false
	validationScope := ""

	if noInput {
		// In non-interactive mode, use the validate flag
		if validateMode != "" && validateMode != "none" {
			shouldValidate = true
			validationScope = validateMode
		}
	} else {
		// Interactive mode - ask the user
		shouldValidate, validationScope = promptForValidation(scanResult)
	}

	if shouldValidate {
		fmt.Printf("\n🤖 Phase 2: AI-powered validation\n")
		fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

		validatedResult, err := runLLMValidation(ctx, scanResult, validationScope, reportDir)
		if err != nil {
			fmt.Printf("⚠️  Warning: LLM validation failed: %v\n", err)
			fmt.Printf("   Pattern scan results are still available.\n")
		} else {
			displayValidationSummary(validatedResult)
			fmt.Printf("\n📁 Validated report saved to: %s\n", reportDir)
		}
	}

	fmt.Printf("\n✅ Scan complete!\n")
	return nil
}

// displayWelcome shows the welcome screen
func displayWelcome() {
	fmt.Printf("\n🔒 PI Scanner - Australian Privacy Compliance\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("Detecting personally identifiable information in code repositories\n")
	fmt.Printf("Press Ctrl+C at any time to exit\n")
}

// promptForValidation asks the user if they want to run LLM validation
func promptForValidation(scanResult *PatternScanResult) (bool, string) {
	// Count findings by risk level
	critical := scanResult.Stats.FindingsByRisk["CRITICAL"]
	high := scanResult.Stats.FindingsByRisk["HIGH"]
	medium := scanResult.Stats.FindingsByRisk["MEDIUM"]
	total := len(scanResult.Findings)

	if total == 0 {
		fmt.Printf("\n✨ No PI findings detected!\n")
		return false, ""
	}

	highRisk := critical + high
	mediumHigh := critical + high + medium

	fmt.Printf("\n📊 Would you like to validate these findings with AI?\n")
	fmt.Printf("This can significantly reduce false positives.\n\n")

	// Show options
	fmt.Printf("1) Validate all findings (%d items) - Est. %d-%d minutes\n", total, total/30, total/20)
	if mediumHigh < total {
		fmt.Printf("2) Validate HIGH + MEDIUM only (%d items) - Est. %d-%d minutes\n", mediumHigh, mediumHigh/30, mediumHigh/20)
	}
	if highRisk < mediumHigh && highRisk > 0 {
		fmt.Printf("3) Validate HIGH + CRITICAL only (%d items) - Est. < %d minute(s)\n", highRisk, (highRisk/20)+1)
	}
	fmt.Printf("4) Skip validation\n\n")

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Choice [2]: ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		input = "2"
	}

	switch input {
	case "1":
		return true, "all"
	case "2":
		return true, "high-medium"
	case "3":
		return true, "high"
	case "4":
		return false, ""
	default:
		// Default to high-medium
		return true, "high-medium"
	}
}
