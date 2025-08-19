# Portfolio Zen Backend

A scalable and reliable Go-based financial API server for portfolio management and stock data.

## 🚀 Features

- **Multiple API Endpoints**: LTP, symbols, health checks, and more
- **Scalable Architecture**: Clean separation of concerns with proper layering
- **Database Connection Pooling**: Optimized PostgreSQL connections
- **Rate Limiting**: Per-IP rate limiting with configurable thresholds
- **Comprehensive Logging**: Structured logging with different levels
- **Health Monitoring**: Health, readiness, and liveness checks
- **Docker Support**: Containerized deployment with docker-compose
- **Middleware Stack**: CORS, authentication, request logging, and more
- **Configuration Management**: Environment-based configuration with validation

## 🏗️ Architecture

```
cmd/server/          # Application entry point
├── main.go         # Main function

internal/            # Internal application code
├── app/            # Application orchestration
├── config/         # Configuration management
├── database/       # Database layer
├── handlers/       # HTTP request handlers
├── logger/         # Logging functionality
├── middleware/     # HTTP middleware
├── responses/      # Response formatting
├── router/         # Route definitions
├── services/       # Business logic services
└── totp/           # TOTP generation utilities
```

## 📋 API Endpoints

### Public Endpoints
- `GET /` - API information
- `GET /api/v1/docs` - API documentation
- `GET /health` - Health check
- `GET /ready` - Readiness check
- `GET /live` - Liveness check

### LTP Endpoints
- `GET /api/v1/ltp/:symbol` - Get LTP for a specific symbol
- `POST /api/v1/ltp/batch` - Get LTP for multiple symbols

### Symbol Endpoints
- `GET /api/v1/symbols` - Get all available symbols
- `GET /api/v1/symbols/search?q=query` - Search symbols

### Protected Endpoints
- `GET /api/v1/protected/profile` - Protected endpoint (requires authentication)

## 🛠️ Setup and Installation

### Prerequisites
- Go 1.25 or higher
- PostgreSQL 15 or higher
- Docker and Docker Compose (optional)

### Local Development

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd portfolio-zen-backend
   ```

2. **Install dependencies**
   ```bash
   go mod download
   go mod tidy
   ```

3. **Set up environment variables**
   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

4. **Run the application**
   ```bash
   make run
   # or
   go run cmd/server/main.go
   ```

### Using Docker

1. **Build and run with docker-compose**
   ```bash
   docker-compose up --build
   ```

2. **Run individual services**
   ```bash
   # Build the application
   make docker-build
   
   # Run the container
   make docker-run
   ```

## 🔧 Configuration

The application uses environment variables for configuration. Key configuration options:

### Server Configuration
- `PORT`: Server port (default: 8000)
- `SERVER_READ_TIMEOUT`: Request read timeout
- `SERVER_WRITE_TIMEOUT`: Response write timeout
- `SERVER_IDLE_TIMEOUT`: Connection idle timeout

### Database Configuration
- `DB_HOST`: Database host
- `DB_PORT`: Database port
- `DB_USER`: Database username
- `DB_PASSWORD`: Database password
- `DB_NAME`: Database name
- `DB_MAX_OPEN_CONNS`: Maximum open connections
- `DB_MAX_IDLE_CONNS`: Maximum idle connections

### Broker Configuration
- `AONE_USERNAME`: Angel One username
- `AONE_PASSWORD`: Angel One password
- `AONE_API_KEY`: Angel One API key
- `AONE_TOKEN`: TOTP secret for authentication

### Rate Limiting
- `RATE_LIMIT_RPM`: Requests per minute
- `RATE_LIMIT_BURST`: Burst limit

## 📊 Database Schema

The application expects the following database tables:

```sql
-- Symbol tokens table
CREATE TABLE symbol_tokens (
    id SERIAL PRIMARY KEY,
    symbol VARCHAR(20) UNIQUE NOT NULL,
    token VARCHAR(20) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Stocks list table (legacy)
CREATE TABLE stocks_list (
    id SERIAL PRIMARY KEY,
    symbol VARCHAR(20) UNIQUE NOT NULL,
    token VARCHAR(20) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes
CREATE INDEX idx_symbol_tokens_symbol ON symbol_tokens(symbol);
CREATE INDEX idx_stocks_list_symbol ON stocks_list(symbol);
```

## 🧪 Testing

Run tests using the Makefile:

```bash
# Run all tests
make test

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/handlers/
```

## 📝 Development

### Available Make Commands

```bash
make help          # Show available commands
make build         # Build the application
make run           # Run the application
make test          # Run tests
make clean         # Clean build artifacts
make lint          # Run linter
make fmt           # Format code
make vet           # Vet code
make deps          # Install dependencies
make dev           # Run with hot reload (requires air)
```

### Code Structure Guidelines

- **Handlers**: Handle HTTP requests and responses
- **Services**: Contain business logic
- **Database**: Handle data persistence
- **Middleware**: Process requests before handlers
- **Config**: Centralize configuration management
- **Logger**: Provide structured logging

### Adding New Endpoints

1. Create a new handler in `internal/handlers/`
2. Add the route in `internal/router/router.go`
3. Update the handler initialization in `internal/app/app.go`
4. Add tests for the new functionality

## 🚀 Deployment

### Production Considerations

- Set `GIN_MODE=release`
- Use proper SSL certificates
- Configure reverse proxy (nginx)
- Set up monitoring and alerting
- Use connection pooling
- Implement proper logging rotation
- Set up backup strategies

### Environment Variables for Production

```bash
GIN_MODE=release
LOG_LEVEL=info
DB_SSLMODE=require
ENABLE_METRICS=true
ENABLE_AUTHENTICATION=true
```

## 📈 Monitoring and Health Checks

The application provides several health check endpoints:

- **`/health`**: Overall application health
- **`/ready`**: Readiness check (is the app ready to serve requests?)
- **`/live`**: Liveness check (is the process running?)

## 🔒 Security Features

- Rate limiting per IP address
- CORS configuration
- Authentication middleware (ready for implementation)
- Request validation
- Error handling without information leakage

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## 📄 License

This project is licensed under the MIT License.

## 🆘 Support

For support and questions, please open an issue in the repository.
