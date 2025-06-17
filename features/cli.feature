Feature: CLI initialization and basic commands
  As a security engineer
  I want to use the PI scanner CLI
  So that I can detect personally identifiable information in repositories

  Background:
    Given the pi-scanner CLI is installed

  Scenario: Display help when run without arguments
    When I run "pi-scanner"
    Then I should see help text containing "PI Scanner - Detect personally identifiable information"
    And I should see available commands including "scan"
    And I should see available commands including "report"
    And I should see available commands including "version"
    And the exit code should be 0

  Scenario: Display version information
    When I run "pi-scanner version"
    Then I should see version information matching pattern "v\d+\.\d+\.\d+"
    And I should see build information
    And the exit code should be 0

  Scenario: Scan a repository with valid URL
    Given a valid GitHub repository URL "https://github.com/test/repo"
    When I run "pi-scanner scan --repo https://github.com/test/repo"
    Then the scan should initiate successfully
    And I should see progress indicators
    And the exit code should be 0

  Scenario: Scan with invalid repository URL
    When I run "pi-scanner scan --repo invalid-url"
    Then I should see an error message "Invalid repository URL"
    And the exit code should be 1

  Scenario: Scan with custom configuration
    Given a configuration file "custom-config.yaml" exists
    When I run "pi-scanner scan --repo https://github.com/test/repo --config custom-config.yaml"
    Then the scan should use the custom configuration
    And I should see "Using configuration: custom-config.yaml"

  Scenario: Scan multiple repositories from file
    Given a file "repos.txt" containing:
      """
      https://github.com/test/repo1
      https://github.com/test/repo2
      """
    When I run "pi-scanner scan --repo-list repos.txt"
    Then the scanner should process 2 repositories
    And I should see progress for each repository

  Scenario: Generate report from previous scan
    Given scan results exist in "scan-results.json"
    When I run "pi-scanner report --input scan-results.json --format html"
    Then an HTML report should be generated
    And I should see "Report generated successfully"
    And the exit code should be 0

  Scenario: Handle interrupt gracefully
    Given a large repository scan is in progress
    When I send SIGINT signal
    Then the scan should stop gracefully
    And I should see "Scan interrupted by user"
    And temporary files should be cleaned up
    And the exit code should be 130