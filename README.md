# Shortly Backend API

A high-performance URL shortening service built with Go, featuring JWT authentication, Redis caching, rate limiting, and comprehensive analytics.

## Features

- URL Shortening with optional custom codes
- JWT-based user authentication
- Click analytics with time-series data
- QR code generation
- URL expiration management
- IP-based rate limiting
- Redis caching for fast lookups
- Database migrations with Goose

## Tech Stack

- Go 1.25+
- Gin web framework
- PostgreSQL
- Redis (optional)
- JWT authentication
- Token Bucket rate limiting

## Installation

1. Clone the repository
   ```bash
   git clone <repository-url>
   cd shortly-be
   ```

2. Install dependencies
   ```bash
   go mod download
   ```

3. Set up environment variables
   Create a `.env` file:
   ```env
   DATABASE_URL=postgres://user:password@localhost:5432/shortly?sslmode=disable
   BASE_URL=http://localhost:8080
   FRONTEND_URL=http://localhost:3000
   REDIS_URL=redis://localhost:6379
   JWT_SECRET=your-super-secret-jwt-key-change-in-production
   JWT_TTL_HOURS=24
   RATE_LIMIT_RPS=10
   RATE_LIMIT_BURST=20
   RATE_LIMIT_AUTH_RPS=5
   RATE_LIMIT_AUTH_BURST=10
   RATE_LIMIT_SHORTEN_RPS=2
   RATE_LIMIT_SHORTEN_BURST=5
   ```

4. Create PostgreSQL database
   ```sql
   CREATE DATABASE shortly;
   ```

5. Run the server
   ```bash
   go run main.go
   ```

Migrations run automatically on server startup. The server starts on `http://localhost:8080`.

## Rate Limiting

The API implements IP-based rate limiting using the Token Bucket algorithm with different limits per endpoint type:

- General API: 10 req/s, burst 20
- Authentication endpoints: 5 req/s, burst 10
- URL shortening: 2 req/s, burst 5
- Redirects: 30 req/s, burst 60

Rate limits are configured via environment variables and can be adjusted per endpoint type. The rate limiter tracks requests per IP address and enforces limits using a token bucket that refills at the specified rate.
