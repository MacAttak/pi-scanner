package main

import (
	"fmt"
	"net/url"
	"os"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

var (
	// Version information (set by build flags)
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "pi-scanner",
		Short: "PI Scanner - Detect personally identifiable information",
		Long: `PI Scanner is a CLI tool for detecting personally identifiable information
in code repositories with a focus on Australian regulatory compliance.

It uses a multi-stage detection pipeline combining pattern matching,
context validation, and algorithmic verification to achieve
high accuracy with minimal false positives.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	// Add commands
	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newScanCmd())
	rootCmd.AddCommand(newReportCmd())

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

func newScanCmd() *cobra.Command {
	var (
		repoURL    string
		repoList   string
		configFile string
		outputFile string
		verbose    bool
	)

	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Scan repositories for personally identifiable information",
		Long: `Scan one or more repositories for personally identifiable information
using a multi-stage detection pipeline.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Validate inputs
			if repoURL == "" && repoList == "" {
				return fmt.Errorf("either --repo or --repo-list must be specified")
			}

			// Validate repository URL format
			if repoURL != "" {
				if err := validateRepositoryURL(repoURL); err != nil {
					return fmt.Errorf("Error: Invalid repository URL: %v", err)
				}
			}

			// Handle repo list
			if repoList != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Reading repository list from: %s\n", repoList)
				// TODO: Implement repo list reading
				return fmt.Errorf("repo list scanning not yet implemented")
			}

			// Single repo scan
			return runScan(cmd.Context(), repoURL, outputFile, verbose)
		},
	}

	// Add flags
	cmd.Flags().StringVarP(&repoURL, "repo", "r", "", "Repository URL to scan")
	cmd.Flags().StringVarP(&repoList, "repo-list", "l", "", "File containing list of repository URLs")
	cmd.Flags().StringVarP(&configFile, "config", "c", "", "Configuration file (default: built-in)")
	cmd.Flags().StringVarP(&outputFile, "output", "o", "scan-results.json", "Output file for results")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")

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

func newReportCmd() *cobra.Command {
	var (
		inputFile  string
		format     string
		outputFile string
	)

	cmd := &cobra.Command{
		Use:   "report",
		Short: "Generate reports from scan results",
		Long:  `Generate HTML, CSV, or SARIF reports from previously saved scan results.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if inputFile == "" {
				return fmt.Errorf("input file must be specified")
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Generating report from: %s\n", inputFile)
			
			if format == "html" {
				fmt.Fprintf(cmd.OutOrStdout(), "Generating HTML report\n")
			}

			// TODO: Implement report generation
			return fmt.Errorf("report generation not yet implemented")
		},
	}

	// Add flags
	cmd.Flags().StringVarP(&inputFile, "input", "i", "", "Input scan results file")
	cmd.Flags().StringVarP(&format, "format", "f", "html", "Report format (html, csv, sarif)")
	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file (default: report.<format>)")

	cmd.MarkFlagRequired("input")

	return cmd
}

// isValidRepoURL performs basic validation of repository URLs
func isValidRepoURL(url string) bool {
	// Basic validation - just check if it starts with https://
	// More comprehensive validation will be added later
	return len(url) > 8 && url[:8] == "https://"
}