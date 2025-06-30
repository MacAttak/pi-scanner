# Security Policy

## Reporting Security Vulnerabilities

The GitHub PI Scanner team takes security vulnerabilities seriously. We appreciate your efforts to responsibly disclose your findings, and will make every effort to acknowledge your contributions.

### How to Report a Vulnerability

**Please do not report security vulnerabilities through public GitHub issues.**

Instead, please report security vulnerabilities by emailing:
- macmilky1@gmail.com

Please include the following information (as much as you can provide):

- Type of issue (e.g., buffer overflow, SQL injection, cross-site scripting, etc.)
- Full paths of source file(s) related to the manifestation of the issue
- The location of the affected source code (tag/branch/commit or direct URL)
- Any special configuration required to reproduce the issue
- Step-by-step instructions to reproduce the issue
- Proof-of-concept or exploit code (if possible)
- Impact of the issue, including how an attacker might exploit the issue

This information will help us triage your report more quickly.

### Preferred Languages

We prefer all communications to be in English.

### Disclosure Policy

When we receive a security bug report, we will:

1. Confirm the problem and determine the affected versions
2. Audit code to find any similar problems
3. Prepare fixes for all releases still under maintenance
4. Release new security fix versions

### Security Update Process

Security updates will be released as soon as possible, and we will:

1. Release patched versions
2. Publish a security advisory on GitHub
3. Credit the reporter (unless they prefer to remain anonymous)

## Supported Versions

We release patches for security vulnerabilities for the following versions:

| Version | Supported          |
| ------- | ------------------ |
| Latest  | :white_check_mark: |
| < 1.0   | :x:                |

## Security Best Practices for Users

When using the GitHub PI Scanner:

1. **Always use local LLM providers** - Never configure the scanner to send PI data to external services
2. **Secure your GitHub tokens** - Use tokens with minimal required permissions
3. **Review scan results carefully** - Ensure masked/redacted outputs before sharing
4. **Run in isolated environments** - Consider using Docker or VMs for scanning untrusted repositories
5. **Keep the scanner updated** - Always use the latest version for security patches

## Security Features

The PI Scanner implements several security measures:

- **Local-only processing** - All scanning happens on your infrastructure
- **Automatic cleanup** - Temporary files are securely deleted after scanning
- **Configurable masking** - PI values can be fully or partially redacted in outputs
- **No data transmission** - The scanner never sends data to external services by default
- **Secure credential handling** - GitHub tokens are never logged or exposed

## Contact

For any security-related questions that don't need to be kept private, feel free to open a discussion in our GitHub repository.
