#!/bin/bash
set -e

echo "Running database migrations..."
./migrator -up

echo "Running database setup script..."
bash ./scripts/db/setup_db.sh

echo "Starting the application..."
exec ./main