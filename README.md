# CCTV Backend Microservice

A Go-based microservice backend for the Flask CCTV surveillance system. This service provides REST API endpoints for managing courts, booking hours, and video clip uploads with PostgreSQL database integration.

## Features

- **Court Management**: Create and retrieve court information
- **Booking Hours**: Manage time slots for court bookings
- **Video Clip Upload**: Handle video file uploads with metadata
- **Database Integration**: PostgreSQL with proper schema and indexing
- **Docker Support**: Full containerization with Docker Compose
- **CORS Support**: Cross-origin resource sharing enabled
- **File Serving**: Static file serving through Nginx
- **Health Check**: Service health monitoring endpoint

## API Endpoints

### Health Check
- `GET /api/v1/health` - Service health status

### Courts
- `GET /api/v1/courts` - Get all courts (with optional name filter)
- `POST /api/v1/courts` - Create a new court

### Booking Hours
- `GET /api/v1/booking-hours` - Get booking hours (with optional courtId filter)
- `POST /api/v1/booking-hours` - Create a new booking hour

### Clips
- `POST /api/v1/clips/upload` - Upload video clip
- `GET /api/v1/clips` - Get clips (with optional bookingHourId filter)

## Quick Start

### Prerequisites
- Docker and Docker Compose
- Go 1.21+ (for local development)
- PostgreSQL 15+ (if running without Docker)

### Using Docker Compose (Recommended)

1. **Clone and setup:**
   ```bash
   git clone <repository-url>
   cd cctv-backend
   cp .env.example .env
   ```

2. **Start all services:**
   ```bash
   make docker-up
   ```
   This will start:
   - PostgreSQL database on port 5432
   - Go backend service on port 5009
   - Nginx reverse proxy on port 80

3. **Check service status:**
   ```bash
   make docker-logs
   curl http://localhost/api/v1/health
   ```

### Local Development

1. **Setup database:**
   ```bash
   # Start only PostgreSQL
   docker-compose up -d postgres
   
   # Or use your local PostgreSQL instance
   createdb cctv_system
   psql cctv_system < schema.sql
   ```

2. **Install dependencies and run:**
   ```bash
   make setup
   make run
   
   # Or for hot reload development
   make install-dev-deps
   make dev
   ```

## Configuration

Environment variables can be set in `.env` file or as system environment variables:

- `DB_HOST` - Database host (default: localhost)
- `DB_PORT` - Database port (default: 5432)
- `DB_USER` - Database user (default: postgres)
- `DB_PASSWORD` - Database password (default: password)
- `DB_NAME` - Database name (default: cctv_system)
- `SERVER_PORT` - Server port (default: 5009)
- `UPLOAD_DIR` - Upload directory (default: ./uploads)

## Database Schema

The service uses PostgreSQL with the following main tables:
- `courts` - Court/camera information
- `booking_hours` - Time slot management
- `clips` - Video clip metadata

Refer to `schema.sql` for complete database structure.

## File Upload

Video clips are uploaded via multipart form data:
- Field name: `video`
- Supported formats: MP4, AVI, WebM
- Max file size: 100MB
- Files are stored in `uploads/clips/` directory

## Integration with Flask App

Update your Flask app's API endpoints to point to this service:

```python
API_BASE_URL = "http://localhost/api/v1"  # or your production URL
CLIP_UPLOAD_ENDPOINT = f"{API_BASE_URL}/clips/upload"
COURTS_ENDPOINT = f"{API_BASE_URL}/courts"
BOOKING_HOURS_ENDPOINT = f"{API_BASE_URL}/booking-hours"
```

## Available Make Commands

- `make help` - Show all available commands
- `make setup` - Setup project dependencies
- `make build` - Build the application
- `make run` - Run locally
- `make test` - Run tests
- `make docker-up` - Start Docker services
- `make docker-down` - Stop Docker services
- `make docker-logs` - View service logs
- `make db-reset` - Reset database
- `make dev` - Hot reload development mode

## Monitoring

### Health Check
```bash
curl http://localhost/api/v1/health
```

### Service Status
```bash
make monitor  # Shows Docker stats and service status
```

### Logs
```bash
make docker-logs
```

## Production Deployment

1. **Update environment variables for production**
2. **Use production compose file:**
   ```bash
   make deploy
   ```
3. **Setup SSL/TLS termination at load balancer level**
4. **Configure proper backup strategy for PostgreSQL**

## Troubleshooting

### Database Connection Issues
```bash
# Check if PostgreSQL is running
docker-compose ps postgres

# Check database logs
docker-compose logs postgres

# Connect to database directly
docker-compose exec postgres psql -U postgres -d cctv_system
```

### Upload Issues
```bash
# Check upload directory permissions
ls -la uploads/

# Check available disk space
df -h

# View backend logs
docker-compose logs backend
```

### Network Issues
```bash
# Test API endpoints
curl -v http://localhost/api/v1/health

# Check nginx configuration
docker-compose exec nginx nginx -t
```

## API Response Format

All API responses follow this format:
```json
{
  "success": true,
  "message": "Operation completed successfully",
  "data": { /* response data */ }
}
```

Error responses:
```json
{
  "success": false,
  "message": "Error description",
  "data": null
}
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make changes and add tests
4. Run `make test` to ensure tests pass
5. Submit a pull request

## License

This project is licensed under the MIT License.