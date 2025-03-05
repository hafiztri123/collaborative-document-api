#!/bin/sh

# Run database migrations
go run scripts/migrate.go -up -config ./config
./scripts/db/setup_db.sh

exec ./out