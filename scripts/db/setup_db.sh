#!/bin/bash

# Load environment variables from .env file if it exists (for local development)
if [ -f .env ]; then
    export $(grep -v '^#' .env | xargs)
fi

# Use Railway's environment variables with fallback defaults
DB_HOST=${PGHOST:-localhost}          # Default to localhost if PGHOST is not set
DB_PORT=${PGPORT:-5432}               # Default to 5432 if PGPORT is not set
DB_NAME=${PGDATABASE:-document_api}   # Default to "document_api" if PGDATABASE is not set
DB_USER=${PGUSER:-postgres}           # Default to "postgres" if PGUSER is not set
DB_PASSWORD=${PGPASSWORD:-postgres}   # Default to "postgres" if PGPASSWORD is not set

echo "Setting up database schema..."
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f scripts/db/schema.sql

if [ $? -eq 0 ]; then
    echo "Database schema setup completed successfully."
else
    echo "Error: Database schema setup failed."
    exit 1
fi