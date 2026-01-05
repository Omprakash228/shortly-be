# Shortly Backend API

A high-performance URL shortening service built with Go, featuring JWT authentication, Redis caching, rate limiting, and comprehensive analytics.

## 🚀 Features

- **URL Shortening**: Create short URLs with optional custom codes
- **User Authentication**: JWT-based authentication with secure token management
- **Click Analytics**: Track clicks with time-series analytics (10min, 30min, 1hr, 6hr, 1day intervals)
- **QR Code Generation**: Generate QR codes for short URLs
- **Expiration Management**: Set expiration dates for URLs with UTC timezone support
- **Rate Limiting**: IP-based rate limiting with configurable limits per endpoint type
- **Redis Caching**: Fast URL lookups and short code availability checks
- **Database Migrations**: Automated schema management with Goose
- **Click Tracking**: Individual click tracking with timestamps

## 🛠 Tech Stack

- **Language**: Go 1.25+
- **Web Framework**: [Gin](https://gin-gonic.com/)
- **Database**: PostgreSQL
- **Cache**: Redis (optional)
- **Authentication**: JWT (JSON Web Tokens)
- **Migrations**: [Goose](https://github.com/pressly/goose)
- **QR Codes**: [go-qrcode](https://github.com/skip2/go-qrcode)
- **Rate Limiting**: Token Bucket algorithm (`golang.org/x/time/rate`)

## 📋 Prerequisites

- Go 1.25 or higher
- PostgreSQL 12+
- Redis (optional, but recommended for production)
- Environment variables configured (see Configuration section)

## 🔧 Installation

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd shortly-be
   ```

2. **Install dependencies**
   ```bash
   go mod download
   ```

3. **Set up environment variables**
   Create a `.env` file in the root directory:
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

4. **Set up PostgreSQL database**
   ```sql
   CREATE DATABASE shortly;
   ```

5. **Run database migrations**
   Migrations run automatically on server startup, but you can also run them manually:
   ```bash
   # Migrations are handled by the application on startup
   # Or use goose CLI if installed:
   goose -dir migrations postgres "your-database-url" up
   ```

## ⚙️ Configuration

### Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `DATABASE_URL` | PostgreSQL connection string | - | ✅ |
| `BASE_URL` | Backend base URL | - | ✅ |
| `FRONTEND_URL` | Frontend base URL (for QR codes) | - | ✅ |
| `REDIS_URL` | Redis connection string | - | ❌ (optional) |
| `JWT_SECRET` | Secret key for JWT signing | - | ✅ |
| `JWT_TTL_HOURS` | JWT token expiration (hours) | 24 | ❌ |
| `RATE_LIMIT_RPS` | General API rate limit (req/s) | 10 | ❌ |
| `RATE_LIMIT_BURST` | General API burst size | 20 | ❌ |
| `RATE_LIMIT_AUTH_RPS` | Auth endpoints rate limit | 5 | ❌ |
| `RATE_LIMIT_AUTH_BURST` | Auth endpoints burst size | 10 | ❌ |
| `RATE_LIMIT_SHORTEN_RPS` | URL shortening rate limit | 2 | ❌ |
| `RATE_LIMIT_SHORTEN_BURST` | URL shortening burst size | 5 | ❌ |

### Rate Limiting

The API implements different rate limits for different endpoint types:

- **General API**: Default 10 req/s, burst 20
- **Authentication**: Stricter 5 req/s, burst 10
- **URL Shortening**: Stricter 2 req/s, burst 5
- **Redirects**: Lenient 30 req/s, burst 60

## 🏃 Running the Server

### Development
```bash
go run main.go
```

### Production
```bash
go build -o shortly-be
./shortly-be
```

The server will start on `http://localhost:8080` by default.

## 📡 API Endpoints

### Public Endpoints

#### Health Check
```
GET /health
```
Returns server status.

#### Redirect
```
GET /:shortCode
```
Redirects to the original URL. Checks expiration before redirecting.

#### Public Redirect (JSON)
```
GET /api/v1/redirect/:shortCode
```
Returns original URL as JSON (used by frontend).

#### QR Code
```
GET /api/v1/qrcode/:shortCode
```
Returns QR code image for the short URL.

### Authentication Endpoints

#### Register
```
POST /api/v1/auth/register
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "securepassword",
  "name": "John Doe"
}
```

#### Login
```
POST /api/v1/auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "securepassword"
}
```

**Response** (both endpoints):
```json
{
  "user_id": "uuid",
  "email": "user@example.com",
  "name": "John Doe",
  "created_at": "2024-01-01T00:00:00Z",
  "token": "jwt-token-here"
}
```

### Protected Endpoints (Require JWT)

All protected endpoints require the `Authorization` header:
```
Authorization: Bearer <jwt-token>
```

#### Create Short URL
```
POST /api/v1/shorten
Authorization: Bearer <token>
Content-Type: application/json

{
  "original_url": "https://example.com",
  "custom_short_code": "mycode",  // optional
  "expires_at": "2024-12-31T23:59:59Z"  // optional, UTC timezone
}
```

**Response**:
```json
{
  "short_code": "mycode",
  "short_url": "http://localhost:3000/mycode",
  "original_url": "https://example.com",
  "expires_at": "2024-12-31T23:59:59Z",
  "created_at": "2024-01-01T00:00:00Z"
}
```

#### Get User URLs
```
GET /api/v1/urls
Authorization: Bearer <token>
```

#### Get URL Stats
```
GET /api/v1/url/:shortCode
Authorization: Bearer <token>
```

**Response**:
```json
{
  "short_code": "abc123",
  "original_url": "https://example.com",
  "click_count": 42,
  "created_at": "2024-01-01T00:00:00Z",
  "expires_at": "2024-12-31T23:59:59Z"
}
```

#### Get Click Analytics
```
GET /api/v1/url/:shortCode/analytics?hours=24
Authorization: Bearer <token>
```

**Query Parameters**:
- `hours`: Time range (6, 12, 24, 72, 168, 336, 720 hours)

**Response**:
```json
[
  {
    "time": "2024-01-01T10:00:00Z",
    "count": 5
  },
  {
    "time": "2024-01-01T11:00:00Z",
    "count": 12
  }
]
```

**Time Grouping**:
- ≤6 hours: 10-minute intervals
- ≤12 hours: 30-minute intervals
- ≤24 hours: 1-hour intervals
- ≤72 hours: 6-hour intervals
- >72 hours: 1-day intervals

#### Update URL Expiration
```
PATCH /api/v1/url/:shortCode
Authorization: Bearer <token>
Content-Type: application/json

{
  "expires_at": "2024-12-31T23:59:59Z"  // UTC timezone
}
```

#### Delete URL
```
DELETE /api/v1/url/:shortCode
Authorization: Bearer <token>
```

## 🏗 Architecture

The project follows a clean architecture pattern:

```
shortly-be/
├── main.go                 # Application entry point
├── migrations/             # Database migrations
├── internal/
│   ├── cache/             # Redis cache implementation
│   ├── config/            # Configuration management
│   ├── controllers/       # HTTP handlers
│   ├── database/          # Database connection & migrations
│   ├── entities/          # Domain entities
│   ├── jwt/               # JWT token generation/validation
│   ├── middleware/        # HTTP middleware (auth, rate limiting)
│   ├── models/            # Request/response DTOs
│   ├── repository/        # Data access layer
│   └── service/           # Business logic layer
```

### Flow

1. **Request** → Middleware (Rate Limiting, Auth)
2. **Controller** → Validates request, extracts data
3. **Service** → Business logic, orchestrates operations
4. **Repository** → Database operations
5. **Cache** → Optional Redis caching layer
6. **Response** → JSON response to client

## 🔐 Authentication

The API uses JWT (JSON Web Tokens) for authentication:

1. User registers/logs in via `/api/v1/auth/register` or `/api/v1/auth/login`
2. Server returns a JWT token
3. Client includes token in `Authorization: Bearer <token>` header
4. `AuthMiddleware` validates token on protected routes
5. User ID is extracted from token and stored in request context

### Token Structure
```json
{
  "user_id": "uuid",
  "email": "user@example.com",
  "exp": 1234567890
}
```

## 💾 Caching Strategy

Redis is used for two purposes:

1. **URL Lookups** (`url:{shortCode}`)
   - Caches full URL data for fast redirects
   - TTL: 1 hour
   - Invalidated on URL deletion/update

2. **Short Code Availability** (`shortcode:exists:{code}`)
   - Caches whether a short code is taken/available
   - TTL: 1 hour for "taken", 30 seconds for "available"
   - Helps prevent duplicate code creation

**Note**: If Redis is unavailable, the application continues without caching (graceful degradation).

## 🗄 Database Schema

### Users Table
- `id` (UUID, Primary Key)
- `email` (VARCHAR, Unique)
- `password_hash` (VARCHAR)
- `name` (VARCHAR)
- `created_at` (TIMESTAMP)

### URLs Table
- `id` (UUID, Primary Key)
- `user_id` (UUID, Foreign Key)
- `original_url` (TEXT)
- `short_code` (VARCHAR, Unique)
- `click_count` (INTEGER)
- `expires_at` (TIMESTAMP WITH TIME ZONE)
- `created_at` (TIMESTAMP)

### URL Clicks Table
- `id` (UUID, Primary Key)
- `url_id` (UUID, Foreign Key)
- `clicked_at` (TIMESTAMP WITH TIME ZONE)

## 🧪 Testing

```bash
# Run tests (if available)
go test ./...

# Test rate limiting
# See test-rate-limit.ps1 for example
```

## 📝 Notes

- All timestamps are stored in UTC in the database
- Expiration checks are performed before redirecting
- Custom short codes are validated (length, format, reserved words)
- Rate limiting is IP-based using Token Bucket algorithm
- Redis is optional - application works without it (slower)

## 🚀 Deployment

1. Set all required environment variables
2. Ensure PostgreSQL and Redis (optional) are accessible
3. Build the application: `go build -o shortly-be`
4. Run migrations (or let the app run them on startup)
5. Start the server: `./shortly-be`

## 📄 License

[Your License Here]

