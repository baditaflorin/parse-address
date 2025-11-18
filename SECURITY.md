# Security Policy

## Overview

This application has been designed with security as a primary concern. This document outlines the security features, best practices, and vulnerability reporting process.

## Security Features

### 1. Input Validation & Sanitization

All user input undergoes strict validation and sanitization:

#### Validation Rules
- **UTF-8 Encoding**: All input must be valid UTF-8
- **Length Limits**: Maximum 10,000 bytes (configurable via `SECURITY_MAX_INPUT_LENGTH`)
- **Null Byte Detection**: Rejects input containing null bytes (`\x00`)
- **Character Filtering**: Removes control characters and dangerous sequences

#### Sanitization Process
```go
// Before processing, all input is:
1. Validated for UTF-8 compliance
2. Checked against length limits
3. Scanned for null bytes
4. Normalized (whitespace, encoding)
5. Truncated to safe limits
```

### 2. Injection Attack Prevention

#### SQL Injection
- **N/A**: This application does not use a database
- No dynamic SQL queries
- No ORM vulnerabilities

#### XSS (Cross-Site Scripting)
- **Content-Type Headers**: Always set to `application/json` for API responses
- **CSP Headers**: Restrictive Content Security Policy
- **Output Encoding**: All output is JSON-encoded
- **No eval()**: No dynamic code execution

#### Command Injection
- **No Shell Execution**: Parser doesn't execute system commands
- **Safe Regex**: Pre-compiled regex patterns only
- **No Dynamic Patterns**: All regex patterns are static

### 3. Denial of Service (DoS) Protection

#### Request Limits
- **Max Request Size**: 1MB default (configurable)
- **Input Length Limits**: 10,000 bytes default
- **Timeout Controls**: Read/write timeouts on all connections

#### Regex Complexity
- **Pre-compiled Patterns**: All regex compiled at startup
- **No Backtracking Issues**: Patterns designed to avoid ReDoS
- **Bounded Execution**: Input limits prevent catastrophic backtracking

#### Rate Limiting
- Configurable via `SECURITY_RATE_LIMIT_PER_MIN`
- Applied at middleware level
- Default: 60 requests per minute per IP

### 4. Web Security Headers

All responses include security headers:

```http
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
Referrer-Policy: strict-origin-when-cross-origin
Content-Security-Policy: default-src 'self' 'unsafe-inline'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'
```

### 5. Container Security

When running in Docker:

#### User Isolation
- **Non-root User**: Runs as user ID 1000 (appuser)
- **No Privileges**: `no-new-privileges:true` flag
- **Read-only Filesystem**: Container filesystem is read-only
- **Minimal Base**: Alpine Linux with minimal packages

#### Network Security
- **Isolated Network**: Custom bridge network
- **Port Restriction**: Only exposes necessary ports
- **Health Checks**: Automated health monitoring

### 6. Configuration Security

#### Environment Variables
- **No Secrets in Code**: All config via environment
- **Validation**: All config values validated on startup
- **Fail-Safe Defaults**: Secure defaults for all settings
- **Clear Error Messages**: Invalid config causes immediate failure

#### Secret Management
```bash
# Never commit secrets to git
.env          # In .gitignore
.env.local    # In .gitignore

# Use secret management tools in production
# - Docker secrets
# - Kubernetes secrets
# - HashiCorp Vault
# - AWS Secrets Manager
```

### 7. Error Handling

#### Safe Error Messages
- **No Stack Traces**: Never expose stack traces to clients
- **Generic Messages**: User sees "Parse error" not internal details
- **Detailed Logs**: Full errors logged server-side only
- **No Information Leakage**: Error messages don't reveal system info

Example:
```go
// ❌ BAD: Exposes internal details
return fmt.Errorf("failed to compile regex at line 42: %v", err)

// ✅ GOOD: Safe generic message
return ErrInvalidInput
```

## Security Testing

### Test Categories

1. **Input Validation Tests**
   - Empty input
   - Oversized input (>10KB)
   - Invalid UTF-8
   - Null bytes
   - Control characters

2. **Injection Attack Tests**
   - SQL injection patterns
   - XSS payloads
   - Command injection attempts
   - Path traversal
   - Format string attacks

3. **DoS Resistance Tests**
   - Very long inputs
   - Repeated patterns
   - Regex complexity attacks
   - Concurrent request floods

4. **Boundary Condition Tests**
   - Minimum/maximum values
   - Edge cases
   - Unicode handling
   - Malformed data

### Running Security Tests

```bash
# Run all security tests
make test-security

# Run with race detector
go test -race ./...

# Run with memory sanitizer
go test -msan ./...

# Fuzz testing
go test -fuzz=FuzzParseAddress -fuzztime=30s ./pkg/parser
```

## Security Tooling

### Recommended Tools

1. **gosec** - Go security checker
```bash
go install github.com/securego/gosec/v2/cmd/gosec@latest
gosec ./...
```

2. **govulncheck** - Vulnerability scanner
```bash
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...
```

3. **nancy** - Dependency vulnerability scanner
```bash
go list -json -m all | nancy sleuth
```

4. **trivy** - Container vulnerability scanner
```bash
trivy image address-parser:latest
```

### CI/CD Integration

```yaml
# Example GitHub Actions workflow
- name: Security Scan
  run: |
    gosec ./...
    govulncheck ./...
    trivy image address-parser:latest
```

## Deployment Security

### Production Checklist

- [ ] Use HTTPS/TLS for all connections
- [ ] Set restrictive CORS origins (not `*`)
- [ ] Enable rate limiting
- [ ] Use strong timeouts
- [ ] Run as non-root user
- [ ] Use read-only filesystem
- [ ] Implement monitoring and alerting
- [ ] Regular security updates
- [ ] Log all security events
- [ ] Use secret management service
- [ ] Enable health checks
- [ ] Implement backup/recovery

### Environment Configuration

```bash
# Production Example
SERVER_HOST=0.0.0.0
SERVER_PORT=8080
SERVER_READ_TIMEOUT=5s
SERVER_WRITE_TIMEOUT=5s
SECURITY_ENABLE_CORS=true
SECURITY_ALLOWED_ORIGINS=https://yourdomain.com
SECURITY_RATE_LIMIT_PER_MIN=100
SECURITY_MAX_INPUT_LENGTH=5000
LOG_LEVEL=warn
LOG_FORMAT=json
```

### TLS/HTTPS

For production, always use a reverse proxy (nginx, Caddy, Traefik) with TLS:

```nginx
server {
    listen 443 ssl http2;
    server_name api.example.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## Vulnerability Reporting

### Reporting Process

If you discover a security vulnerability:

1. **DO NOT** open a public GitHub issue
2. Email security details to: [security contact email]
3. Include:
   - Description of vulnerability
   - Steps to reproduce
   - Potential impact
   - Suggested fix (if any)

### Response Timeline

- **24 hours**: Initial response acknowledging report
- **72 hours**: Preliminary assessment
- **7 days**: Patch development (for confirmed vulnerabilities)
- **14 days**: Public disclosure (after patch release)

### Hall of Fame

Security researchers who responsibly disclose vulnerabilities will be credited here (with permission).

## Security Audit History

| Date | Auditor | Scope | Status |
|------|---------|-------|--------|
| 2025-01 | Internal | Initial security review | ✅ Passed |

## Compliance

### OWASP Top 10 (2021)

| Risk | Status | Mitigation |
|------|--------|------------|
| A01: Broken Access Control | ✅ | No authentication required, public API |
| A02: Cryptographic Failures | ✅ | No sensitive data storage |
| A03: Injection | ✅ | Input validation, no SQL/command execution |
| A04: Insecure Design | ✅ | Security-first architecture |
| A05: Security Misconfiguration | ✅ | Secure defaults, validation |
| A06: Vulnerable Components | ✅ | Regular updates, scanning |
| A07: Auth Failures | N/A | No authentication |
| A08: Data Integrity | ✅ | No data persistence |
| A09: Logging Failures | ✅ | Comprehensive logging |
| A10: SSRF | ✅ | No external requests |

## License

This security policy is part of the address parser project and follows the same ISC license.

## Updates

This security policy is reviewed and updated quarterly. Last update: 2025-01-18
