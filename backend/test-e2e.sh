#!/bin/bash
set -e

echo "=== End-to-End Test ==="

# Check prerequisites
if [ -z "$CF_API_URL" ]; then
  echo "Error: CF_API_URL not set"
  exit 1
fi

if [ -z "$CF_USERNAME" ]; then
  echo "Error: CF_USERNAME not set"
  exit 1
fi

if [ -z "$CF_PASSWORD" ]; then
  echo "Error: CF_PASSWORD not set"
  exit 1
fi

# Build backend
echo "Building backend..."
go build -o capacity-backend

# Start backend in background
echo "Starting backend..."
./capacity-backend &
BACKEND_PID=$!
sleep 2

# Cleanup on exit
cleanup() {
  echo "Stopping backend..."
  kill $BACKEND_PID 2>/dev/null || true
  rm -f capacity-backend
}
trap cleanup EXIT

# Test health endpoint
echo "Testing /api/health..."
HEALTH=$(curl -s http://localhost:8080/api/health)
echo "$HEALTH" | jq .

if ! echo "$HEALTH" | jq -e '.cf_api == "ok"' > /dev/null; then
  echo "Error: Health check failed"
  exit 1
fi

# Test dashboard endpoint
echo "Testing /api/dashboard..."
DASHBOARD=$(curl -s http://localhost:8080/api/dashboard)
echo "$DASHBOARD" | jq .

if ! echo "$DASHBOARD" | jq -e '.metadata.timestamp' > /dev/null; then
  echo "Error: Dashboard response invalid"
  exit 1
fi

echo "=== All tests passed ==="
