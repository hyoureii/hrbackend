#!/bin/sh
set -e

export PGHOST="$POSTGRES_HOST"
export PGPORT="$POSTGRES_PORT"
export PGUSER="$POSTGRES_USER"
export PGPASSWORD="$POSTGRES_PASSWORD"
export PGDATABASE="$POSTGRES_DBNAME"

echo "Waiting for PostgreSQL..."
until pg_isready -q; do
  sleep 1
done

echo "Running migrations..."
/migrate push

echo "Seeding database..."
/seed

echo "Starting server..."
exec /server
