#!/bin/bash
export PORT=${PORT:-8080}
export HOST=${HOST:-0.0.0.0}
export DATABASE_URL=${DATABASE_URL:-"file:./alchemorsel_dev.db"}
export JWT_SECRET=${JWT_SECRET:-"your-secret-key"}

./main
