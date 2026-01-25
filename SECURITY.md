# Security Policy

## Supported Versions

We release patches for security vulnerabilities for the following versions:

| Version | Supported          |
| ------- | ------------------ |
| latest  | :white_check_mark: |
| < 1.0   | :x:                |

## Reporting a Vulnerability

We take the security of apimgr seriously. If you believe you have found a security vulnerability, please report it to us as described below.

### Where to Report

**Please do NOT report security vulnerabilities through public GitHub issues.**

Instead, please report them via:
- Email: [security contact needed - replace with actual email]
- GitHub Security Advisory: https://github.com/ccasJay/apimgr/security/advisories/new

### What to Include

Please include the following information in your report:
- Type of vulnerability
- Full paths of source file(s) related to the vulnerability
- Location of the affected source code (tag/branch/commit or direct URL)
- Step-by-step instructions to reproduce the issue
- Proof-of-concept or exploit code (if possible)
- Impact of the issue, including how an attacker might exploit it

### Response Timeline

- We will acknowledge receipt of your vulnerability report within 48 hours
- We will send you a more detailed response within 7 days indicating the next steps
- We will keep you informed of the progress towards a fix and full announcement
- We may ask for additional information or guidance

## Security Best Practices

### For Users

When using apimgr, follow these security best practices:

1. **Protect Your API Keys**
   - Never commit configuration files containing API keys to version control
   - Use `.gitignore` to exclude `~/.config/apimgr/` from any repositories
   - Regularly rotate your API keys

2. **File Permissions**
   - apimgr automatically sets configuration files to 0600 (owner read/write only)
   - Verify permissions: `ls -la ~/.config/apimgr/config.json`
   - Do not modify these permissions to be more permissive

3. **Environment Variables**
   - Be cautious when using environment variables in shared systems
   - Use local switch (`-l` flag) for temporary configurations
   - Clear environment variables when no longer needed

4. **Shell Integration**
   - Review the `active.env` file before sourcing it
   - Use shell-specific security features (e.g., read-only variables)

5. **Network Security**
   - Always use HTTPS URLs for API endpoints
   - Verify SSL/TLS certificates are valid
   - Use network monitoring to detect unusual API activity

### For Developers

If you're contributing to apimgr:

1. **Code Review**
   - All changes must be reviewed before merging
   - Security-sensitive changes require additional review

2. **Dependencies**
   - Keep dependencies up to date
   - Regularly check for known vulnerabilities with `go list -m all | nancy sleuth`
   - Review dependency changes in pull requests

3. **Sensitive Data**
   - Never log API keys or tokens
   - Use masking functions (e.g., `MaskAPIKey`) when displaying sensitive data
   - Avoid storing sensitive data in memory longer than necessary

4. **Input Validation**
   - Validate all user inputs
   - Sanitize inputs used in system calls
   - Use parameterized queries/commands

## Security Features

apimgr includes several built-in security features:

### API Key Protection
- **Masking**: API keys are masked in display output (e.g., `sk-ant-****...`)
- **File Permissions**: Configuration files are automatically set to 0600
- **No Logging**: API keys are never logged to files or console

### File Locking
- Prevents concurrent modifications to configuration files
- Uses platform-specific file locking mechanisms
- Automatically releases locks on error or completion

### Input Validation
- URL format validation
- Required field checks
- Special character filtering

### Secure Storage
- Configuration stored in XDG-compliant directory
- Optional encryption support (planned for future release)
- Automatic migration from legacy locations

## Known Limitations

### Current Security Considerations

1. **Plaintext Storage**: API keys are currently stored in plaintext in JSON configuration files. While file permissions are restrictive (0600), users on shared systems should be aware of this limitation.

2. **Environment Variables**: When using shell integration, API keys are exported as environment variables and may be visible to other processes running under the same user.

3. **Memory**: API keys are held in memory during program execution. While this is standard practice, it means keys could potentially be read from memory dumps.

### Planned Security Enhancements

- **Optional Encryption**: Add support for encrypting configuration files at rest
- **Secure Key Storage**: Integration with system keychains/credential managers
- **Audit Logging**: Log all configuration changes and API usage
- **MFA Support**: Multi-factor authentication for sensitive operations
- **Key Rotation**: Automated API key rotation support

## Security Disclosure Policy

When we receive a security bug report, we will:

1. Confirm the problem and determine the affected versions
2. Audit code to find any potential similar problems
3. Prepare fixes for all supported versions
4. Release new security fix versions as soon as possible

## Attribution

We appreciate the security research community and believe in responsible disclosure. Security researchers who report valid vulnerabilities will be acknowledged in:
- Release notes for security fixes
- Security advisories
- This SECURITY.md file (with permission)

## Questions

If you have questions about this security policy, please contact us through GitHub Issues (for non-sensitive questions) or via the vulnerability reporting channels above (for sensitive questions).

---

Last Updated: 2025-01-25
