#!/bin/bash
set -e  # Exit immediately if a command exits with a non-zero status

echo "Running database migrations..."
go run ./migrate.go -up

echo "Running database setup script..."
bash ./setup_db.sh

echo "Starting the application..."
exec ./out