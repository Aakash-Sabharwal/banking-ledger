#!/bin/bash

# Banking Ledger Service Local Startup with Homebrew Services

echo "Starting Banking Ledger Service with Homebrew services..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Check if services are running
check_services() {
    print_status "Checking if services are running..."
    
    # Check PostgreSQL
    if /opt/homebrew/opt/postgresql@15/bin/pg_isready -h localhost -p 5432 > /dev/null 2>&1; then
        print_success "PostgreSQL is running"
    else
        print_error "PostgreSQL is not running. Starting it..."
        brew services start postgresql@15
        sleep 3
    fi
    
    # Check MongoDB
    if mongosh --eval "db.runCommand('ping')" --quiet > /dev/null 2>&1; then
        print_success "MongoDB is running"
    else
        print_error "MongoDB is not running. Starting it..."
        brew services start mongodb/brew/mongodb-community
        sleep 3
    fi
    
    # Check RabbitMQ
    if curl -s http://localhost:15672 > /dev/null 2>&1; then
        print_success "RabbitMQ is running"
    else
        print_error "RabbitMQ is not running. Starting it..."
        brew services start rabbitmq
        sleep 5
    fi
}

# Set environment variables for local Homebrew services
export DATABASE_URL="postgres://$(whoami)@localhost:5432/banking_ledger?sslmode=disable"
export MONGODB_URL="mongodb://localhost:27017/ledger"
export RABBITMQ_URL="amqp://guest:guest@localhost:5672/"
export SERVER_PORT="8080"
export LOG_LEVEL="info"

print_status "Environment variables set:"
print_status "DATABASE_URL: $DATABASE_URL"
print_status "MONGODB_URL: $MONGODB_URL"
print_status "RABBITMQ_URL: $RABBITMQ_URL"
print_status "SERVER_PORT: $SERVER_PORT"

# Check and start services
check_services

# Wait a bit for services to stabilize
print_status "Waiting for services to stabilize..."
sleep 5

# Build applications if they don't exist
if [[ ! -f "./bin/api" || ! -f "./bin/processor" ]]; then
    print_status "Building applications..."
    mkdir -p bin
    
    if go build -o bin/api ./cmd/api && go build -o bin/processor ./cmd/processor; then
        print_success "Applications built successfully"
    else
        print_error "Failed to build applications"
        exit 1
    fi
fi

print_status "Starting API service..."
./bin/api &
API_PID=$!

print_status "Starting processor service..."
./bin/processor &
PROCESSOR_PID=$!

# Wait a moment for services to start
sleep 3

echo
print_success "Services started!"
echo "API running at: http://localhost:8080"
echo "Health check: http://localhost:8080/health"
echo "API docs: http://localhost:8080/api/v1/docs"
echo "RabbitMQ Management: http://localhost:15672 (guest/guest)"
echo "PostgreSQL connection: psql -d banking_ledger"
echo "MongoDB connection: mongosh ledger"
echo ""
echo "Press Ctrl+C to stop all services"

# Test API health
sleep 2
if curl -s http://localhost:8080/health > /dev/null 2>&1; then
    print_success "API health check passed!"
else
    print_warning "API health check failed - check logs above"
fi

# Cleanup function
cleanup() {
    echo ""
    print_warning "Stopping services..."
    kill $API_PID $PROCESSOR_PID 2>/dev/null || true
    print_success "Services stopped"
    exit 0
}

# Set trap for cleanup
trap cleanup SIGINT SIGTERM

# Wait for services
wait