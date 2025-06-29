Feature: CLI initialization and basic commands
  As a security engineer
  I want to use the PI scanner CLI
  So that I can detect personally identifiable information in repositories

  Background:
    Given the pi-scanner CLI is installed

  Scenario: Display help when run without arguments
    When I run "pi-scanner"
    Then I should see help text containing "PI Scanner - Australian Privacy Compliance"
    And I should see available commands including "llm-check"
    And I should see available commands including "version"
    And the exit code should be 0

  Scenario: Display version information
    When I run "pi-scanner version"
    Then I should see version information matching pattern "v\d+\.\d+\.\d+"
    And I should see build information
    And the exit code should be 0

  Scenario: Scan a repository with valid URL
    Given a valid GitHub repository URL "https://github.com/test/repo"
    When I run "pi-scanner https://github.com/test/repo --no-input"
    Then the scan should initiate successfully
    And I should see progress indicators
    And the exit code should be 0

  Scenario: Scan with invalid repository URL
    When I run "pi-scanner invalid-url"
    Then I should see an error message "invalid repository URL"
    And the exit code should be 1

  Scenario: Scan with masking level
    Given a valid GitHub repository URL "https://github.com/test/repo"
    When I run "pi-scanner https://github.com/test/repo --no-input --masking=full"
    Then the scan should complete with full masking applied
    And the exit code should be 0

  Scenario: Scan with validation mode
    Given a valid GitHub repository URL "https://github.com/test/repo"
    When I run "pi-scanner https://github.com/test/repo --no-input --validate=high"
    Then the scan should complete with high-risk validation
    And the exit code should be 0

  Scenario: Check LLM service availability
    When I run "pi-scanner llm-check"
    Then I should see "LLM Service Check"
    And I should see "Checking service availability"
    And the exit code should be 0 or 1 depending on service status

  Scenario: Handle interrupt gracefully
    Given a large repository scan is in progress
    When I send SIGINT signal
    Then the scan should stop gracefully
    And I should see "Scan interrupted by user"
    And temporary files should be cleaned up
    And the exit code should be 130
