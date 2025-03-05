FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o api ./cmd/api

FROM alpine:latest

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/api .
COPY --from=builder /app/config ./config
COPY --from=builder /app/scripts ./scripts

# Expose the application port
EXPOSE 8080

# Run the application
CMD ["./entrypoint.sh"]