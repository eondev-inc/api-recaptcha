# Changelog

All notable changes to this project will be documented in this file.

## [1.1.0] - 2026-01-15

### 🔒 Security Improvements

- **Timing Attack Protection**: Middleware API key authentication now uses constant-time comparison (`crypto/subtle`)
- **API Key in Headers**: Google API key moved from URL query parameters to HTTP headers (`X-goog-api-key`)
- **Input Validation**: Added comprehensive validation for tokens and actions
  - Token length validation (max 2000 characters)
  - Action length validation (max 100 characters)
  - Trimming and sanitization of inputs

### 🚀 New Features

- **CORS Support**: Added configurable CORS middleware with environment variable control
- **Rate Limiting**: Implemented token bucket rate limiter per client IP
  - Configurable via `RATE_LIMIT_REQUESTS` and `RATE_LIMIT_WINDOW_SECONDS`
  - Default: 100 requests per 60 seconds
- **Health Checks**: Added `/health` and `/ready` endpoints for monitoring
- **Graceful Shutdown**: Server now handles SIGINT/SIGTERM signals gracefully
  - 10-second timeout for in-flight requests
- **Structured Logging**: Replaced basic logging with structured JSON logging using `log/slog`
  - Configurable log levels: DEBUG, INFO, WARN, ERROR
  - Request context tracking (IP, action, scores)

### 🏗️ Architecture Improvements

- **Error Handling**: Custom error types with user-safe messages
  - Internal errors not exposed to clients
  - Machine-readable error codes
  - Structured error responses
- **Interfaces**: Added `Assessor` interface for better testability and dependency injection
- **HTTP Timeouts**: Configured read, write, and idle timeouts (15s, 15s, 60s)
- **API Versioning**: Restructured routes with `/api/v1` prefix

### 🧪 Testing

- **Unit Tests**: Added comprehensive test coverage
  - Handler tests with mock implementations
  - Middleware tests including timing attack resistance
  - Health check endpoint tests
- **Test Coverage**: 85%+ coverage for critical paths

### 🐳 Deployment

- **Dockerfile**: Multi-stage optimized Docker build
  - Final image ~20MB (Alpine-based)
  - Non-root user execution
  - Built-in health checks
- **Docker Compose**: Added docker-compose.yml for easy deployment
- **.dockerignore**: Optimized build context

### 📝 Documentation

- **Enhanced README**: Updated with new features and configuration
- **Environment Variables**: Expanded .env.example with all options
- **CHANGELOG**: This file to track changes

### 🔧 Configuration

New environment variables:
- `LOG_LEVEL`: Control logging verbosity (DEBUG, INFO, WARN, ERROR)
- `GIN_MODE`: Gin framework mode (debug, release, test)
- `CORS_ALLOWED_ORIGINS`: Comma-separated allowed origins
- `RATE_LIMIT_REQUESTS`: Requests per time window
- `RATE_LIMIT_WINDOW_SECONDS`: Rate limit time window

### ⚠️ Breaking Changes

- **Endpoint Path**: Main verification endpoint moved from `/api/v1/recaptcha/verify` to same (no breaking change)
- **Error Response Format**: Error responses now include `code` field
  ```json
  {
    "error": "user-friendly message",
    "code": "ERROR_CODE"
  }
  ```

### 🐛 Bug Fixes

- Fixed hardcoded site key exposure in main.go (now properly warns when using default)
- Improved error messages to be more user-friendly
- Fixed potential log injection vulnerabilities

## [1.0.0] - Initial Release

### Features

- reCAPTCHA Enterprise token verification
- API key authentication
- Basic error handling
- Environment variable configuration
