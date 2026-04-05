# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 1.0.x   | :white_check_mark: |

## Reporting a Vulnerability

If you discover a security vulnerability, please send an email to the maintainer. All security vulnerabilities will be promptly addressed.

Please include the following information:
- Type of vulnerability
- Full paths of source file(s) related to the vulnerability
- Location of the affected source code
- Any special configuration required to reproduce the issue
- Step-by-step instructions to reproduce the issue
- Proof-of-concept or exploit code (if possible)
- Impact of the issue, including how an attacker might exploit it

## Security Best Practices

- Use Ed25519 for cryptographic signatures (NIST Curve25519)
- Use crypto/rand for random number generation
- All SQL queries use parameterized statements
- HTTP client has 10-second timeout
- Docker image uses distroless/static:nonroot (no shell)