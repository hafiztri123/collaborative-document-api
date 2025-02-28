# Build stage
FROM golang:1.21-alpine AS build

WORKDIR /app

# Install dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app ./cmd/api

# Final stage
FROM alpine:latest

WORKDIR /app

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Copy the binary from the build stage
COPY --from=build /app/app .
COPY --from=build /app/config ./config
COPY --from=build /app/migrations ./migrations

# Set environment variables
ENV GIN_MODE=release

# Expose port
EXPOSE 8080

# Run the application
CMD ["./app"]