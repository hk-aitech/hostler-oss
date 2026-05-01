# Security Review Checklist

## OWASP Top 10 Checks

### A01: Broken Access Control
- [ ] Authorization checks on all protected endpoints
- [ ] Deny by default policy
- [ ] CORS properly configured
- [ ] Directory listing disabled

### A02: Cryptographic Failures
- [ ] No hardcoded secrets or API keys
- [ ] Sensitive data encrypted at rest
- [ ] Strong encryption algorithms (AES-256, RSA-2048+)
- [ ] TLS 1.2+ for data in transit

### A03: Injection
- [ ] Parameterized queries for SQL
- [ ] Input validation on all user inputs
- [ ] Output encoding for HTML/JS
- [ ] Command injection prevention

### A04: Insecure Design
- [ ] Threat modeling completed
- [ ] Security requirements defined
- [ ] Defense in depth applied
- [ ] Fail securely

### A05: Security Misconfiguration
- [ ] Default credentials changed
- [ ] Unnecessary features disabled
- [ ] Error messages don't leak info
- [ ] Security headers configured

### A06: Vulnerable Components
- [ ] Dependencies up to date
- [ ] No known vulnerabilities
- [ ] Components from trusted sources
- [ ] Automated vulnerability scanning

### A07: Authentication Failures
- [ ] Strong password policy
- [ ] Multi-factor authentication
- [ ] Session timeout configured
- [ ] Brute force protection

### A08: Data Integrity Failures
- [ ] Integrity checks on updates
- [ ] Signed packages/updates
- [ ] CI/CD pipeline security
- [ ] Deserialization safety

### A09: Logging Failures
- [ ] Security events logged
- [ ] No sensitive data in logs
- [ ] Log integrity protected
- [ ] Alerting configured

### A10: SSRF
- [ ] URL validation
- [ ] Allowlist for external calls
- [ ] Network segmentation
- [ ] Response validation

## Detection Patterns

### Hardcoded Secrets
```bash
grep -rn --include="*.{cs,ts,js,py,go,java}" \
  -E "(password|secret|api_key|token|credential)\s*=\s*['\"][^'\"]+['\"]" .
```

### SQL Injection Risk
```bash
grep -rn --include="*.{cs,ts,js,py,go,java}" \
  -E "(SELECT|INSERT|UPDATE|DELETE).*\+.*\"|f\".*SELECT" .
```

### Sensitive Data Exposure
```bash
grep -rn --include="*.{cs,ts,js,py,go,java}" \
  -E "Log\.(Information|Debug|Warning|Error).*password|Log.*apiKey" .
```
