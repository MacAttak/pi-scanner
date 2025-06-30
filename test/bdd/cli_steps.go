package bdd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/cucumber/godog"
)

type cliContext struct {
	binaryPath  string
	lastCommand *exec.Cmd
	stdout      *bytes.Buffer
	stderr      *bytes.Buffer
	exitCode    int
	tempDir     string
	interrupted bool
}

var cli = &cliContext{}

func registerCLISteps(ctx *godog.ScenarioContext) {
	// Background
	ctx.Step(`^the pi-scanner CLI is installed$`, ensureCLIInstalled)

	// Execution steps
	ctx.Step(`^I run "([^"]*)"$`, iRunCommand)
	ctx.Step(`^a valid GitHub repository URL "([^"]*)"$`, aValidGitHubRepositoryURL)
	ctx.Step(`^a large repository scan is in progress$`, aLargeRepositoryScanIsInProgress)
	ctx.Step(`^I send SIGINT signal$`, iSendSIGINTSignal)

	// Assertion steps
	ctx.Step(`^I should see help text containing "([^"]*)"$`, iShouldSeeHelpTextContaining)
	ctx.Step(`^I should see available commands including "([^"]*)"$`, iShouldSeeAvailableCommandsIncluding)
	ctx.Step(`^I should see version information matching pattern "([^"]*)"$`, iShouldSeeVersionInformationMatchingPattern)
	ctx.Step(`^I should see build information$`, iShouldSeeBuildInformation)
	ctx.Step(`^I should see an error message "([^"]*)"$`, iShouldSeeAnErrorMessage)
	ctx.Step(`^I should see "([^"]*)"$`, iShouldSee)
	ctx.Step(`^I should see progress indicators$`, iShouldSeeProgressIndicators)
	ctx.Step(`^the exit code should be (\d+)$`, theExitCodeShouldBe)
	ctx.Step(`^the exit code should be (\d+) or (\d+) depending on service status$`, theExitCodeShouldBeOrDependingOnServiceStatus)

	// Completion steps
	ctx.Step(`^the scan should initiate successfully$`, theScanShouldInitiateSuccessfully)
	ctx.Step(`^the scan should complete with full masking applied$`, theScanShouldCompleteWithFullMaskingApplied)
	ctx.Step(`^the scan should complete with high-risk validation$`, theScanShouldCompleteWithHighRiskValidation)
	ctx.Step(`^the scan should stop gracefully$`, theScanShouldStopGracefully)
	ctx.Step(`^temporary files should be cleaned up$`, temporaryFilesShouldBeCleanedUp)

	// Cleanup
	ctx.After(func(ctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
		cleanupCLI()
		return ctx, nil
	})
}

func ensureCLIInstalled() error {
	// Build the binary if not already built
	if cli.binaryPath == "" {
		buildCmd := exec.Command("go", "build", "-o", "pi-scanner-test", "./cmd/pi-scanner")
		buildCmd.Dir = filepath.Join("..", "..")
		if err := buildCmd.Run(); err != nil {
			return fmt.Errorf("failed to build CLI: %w", err)
		}
		cli.binaryPath = filepath.Join("..", "..", "pi-scanner-test")
	}

	// Create temp directory for test outputs
	tempDir, err := os.MkdirTemp("", "pi-scanner-bdd-*")
	if err != nil {
		return err
	}
	cli.tempDir = tempDir

	return nil
}

func iRunCommand(command string) error {
	// Replace pi-scanner with actual binary path
	args := strings.Fields(strings.Replace(command, "pi-scanner", cli.binaryPath, 1))

	cli.lastCommand = exec.Command(args[0], args[1:]...)
	cli.stdout = &bytes.Buffer{}
	cli.stderr = &bytes.Buffer{}
	cli.lastCommand.Stdout = cli.stdout
	cli.lastCommand.Stderr = cli.stderr
	cli.lastCommand.Dir = cli.tempDir

	err := cli.lastCommand.Run()
	if exitErr, ok := err.(*exec.ExitError); ok {
		cli.exitCode = exitErr.ExitCode()
	} else if err != nil {
		return fmt.Errorf("failed to run command: %w", err)
	} else {
		cli.exitCode = 0
	}

	return nil
}

func aValidGitHubRepositoryURL(url string) error {
	// This is a setup step, no action needed
	return nil
}

func aLargeRepositoryScanIsInProgress() error {
	// Start a scan in the background
	args := []string{cli.binaryPath, "https://github.com/MacAttak/pi-scanner-test-data", "--no-input"}
	cli.lastCommand = exec.Command(args[0], args[1:]...)
	cli.stdout = &bytes.Buffer{}
	cli.stderr = &bytes.Buffer{}
	cli.lastCommand.Stdout = cli.stdout
	cli.lastCommand.Stderr = cli.stderr

	if err := cli.lastCommand.Start(); err != nil {
		return err
	}

	// Give it a moment to start
	time.Sleep(2 * time.Second)

	return nil
}

func iSendSIGINTSignal() error {
	if cli.lastCommand == nil || cli.lastCommand.Process == nil {
		return fmt.Errorf("no process running")
	}

	cli.interrupted = true
	return cli.lastCommand.Process.Signal(os.Interrupt)
}

func iShouldSeeHelpTextContaining(text string) error {
	output := cli.stdout.String()
	if !strings.Contains(output, text) {
		return fmt.Errorf("expected output to contain %q, got: %s", text, output)
	}
	return nil
}

func iShouldSeeAvailableCommandsIncluding(command string) error {
	output := cli.stdout.String()
	if !strings.Contains(output, command) {
		return fmt.Errorf("expected to see command %q in output, got: %s", command, output)
	}
	return nil
}

func iShouldSeeVersionInformationMatchingPattern(pattern string) error {
	output := cli.stdout.String()
	matched, err := regexp.MatchString(pattern, output)
	if err != nil {
		return err
	}
	if !matched {
		return fmt.Errorf("output does not match pattern %q, got: %s", pattern, output)
	}
	return nil
}

func iShouldSeeBuildInformation() error {
	output := cli.stdout.String()
	if !strings.Contains(output, "Build") && !strings.Contains(output, "Commit") {
		return fmt.Errorf("expected build information in output, got: %s", output)
	}
	return nil
}

func iShouldSeeAnErrorMessage(message string) error {
	output := cli.stderr.String() + cli.stdout.String()
	if !strings.Contains(strings.ToLower(output), strings.ToLower(message)) {
		return fmt.Errorf("expected error message %q, got: %s", message, output)
	}
	return nil
}

func iShouldSee(text string) error {
	output := cli.stdout.String()
	if !strings.Contains(output, text) {
		return fmt.Errorf("expected to see %q in output, got: %s", text, output)
	}
	return nil
}

func iShouldSeeProgressIndicators() error {
	output := cli.stdout.String()
	// Look for common progress indicators
	progressPatterns := []string{"Progress", "Scanning", "%", "files", "[", "]"}
	found := false
	for _, pattern := range progressPatterns {
		if strings.Contains(output, pattern) {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("no progress indicators found in output: %s", output)
	}
	return nil
}

func theExitCodeShouldBe(expected int) error {
	if cli.exitCode != expected {
		return fmt.Errorf("expected exit code %d, got %d", expected, cli.exitCode)
	}
	return nil
}

func theExitCodeShouldBeOrDependingOnServiceStatus(code1, code2 int) error {
	if cli.exitCode != code1 && cli.exitCode != code2 {
		return fmt.Errorf("expected exit code %d or %d, got %d", code1, code2, cli.exitCode)
	}
	return nil
}

func theScanShouldInitiateSuccessfully() error {
	if cli.exitCode != 0 {
		return fmt.Errorf("scan did not initiate successfully, exit code: %d", cli.exitCode)
	}
	output := cli.stdout.String()
	if !strings.Contains(output, "Scanning") && !strings.Contains(output, "Phase 1") {
		return fmt.Errorf("scan output does not indicate successful initiation: %s", output)
	}
	return nil
}

func theScanShouldCompleteWithFullMaskingApplied() error {
	if cli.exitCode != 0 {
		return fmt.Errorf("scan did not complete successfully, exit code: %d", cli.exitCode)
	}
	// Check that masking was applied
	output := cli.stdout.String()
	if strings.Contains(output, "masking=full") || strings.Contains(output, "Masking: full") {
		return nil
	}
	return fmt.Errorf("full masking not confirmed in output: %s", output)
}

func theScanShouldCompleteWithHighRiskValidation() error {
	if cli.exitCode != 0 {
		return fmt.Errorf("scan did not complete successfully, exit code: %d", cli.exitCode)
	}
	output := cli.stdout.String()
	if strings.Contains(output, "validate=high") || strings.Contains(output, "Validation: high") {
		return nil
	}
	return fmt.Errorf("high-risk validation not confirmed in output: %s", output)
}

func theScanShouldStopGracefully() error {
	// Wait for process to finish after interrupt
	cli.lastCommand.Wait()

	output := cli.stdout.String() + cli.stderr.String()
	if !strings.Contains(output, "interrupt") && !strings.Contains(output, "Interrupt") {
		return fmt.Errorf("no interrupt message found in output: %s", output)
	}
	return nil
}

func temporaryFilesShouldBeCleanedUp() error {
	// Check common temp directories
	tempDirs := []string{"/tmp", os.TempDir()}
	for _, dir := range tempDirs {
		files, _ := filepath.Glob(filepath.Join(dir, "pi-scanner-*"))
		if len(files) > 0 {
			return fmt.Errorf("found %d temporary files that weren't cleaned up", len(files))
		}
	}
	return nil
}

func cleanupCLI() {
	if cli.tempDir != "" {
		os.RemoveAll(cli.tempDir)
	}
	if cli.binaryPath != "" && strings.Contains(cli.binaryPath, "test") {
		os.Remove(cli.binaryPath)
	}
}
