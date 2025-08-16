FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY *.go ./

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Final stage
FROM alpine:latest

# Install necessary packages
RUN apk --no-cache add ca-certificates tzdata postgresql-client

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/main .

# Create upload directories
RUN mkdir -p /app/uploads/clips

# Copy wait script
COPY wait-for-db.sh .
RUN chmod +x wait-for-db.sh

EXPOSE 5009

# Use wait script to ensure database is ready
CMD ["./wait-for-db.sh", "./main"]