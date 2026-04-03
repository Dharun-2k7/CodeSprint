#!/bin/bash

# Codesprint Setup Script
# This script automates the initial setup of the Codesprint platform

set -e  # Exit on error

echo "============================================"
echo "  Codesprint Setup Script"
echo "============================================"
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_info() {
    echo -e "${YELLOW}ℹ $1${NC}"
}

# Check if Docker is installed
print_info "Checking Docker installation..."
if ! command -v docker &> /dev/null; then
    print_error "Docker is not installed. Please install Docker first."
    exit 1
fi
print_success "Docker is installed"

# Check if Docker Compose is installed
print_info "Checking Docker Compose installation..."
if ! command -v docker-compose &> /dev/null; then
    print_error "Docker Compose is not installed. Please install Docker Compose first."
    exit 1
fi
print_success "Docker Compose is installed"

# Create .env file if it doesn't exist
if [ ! -f .env ]; then
    print_info "Creating .env file..."
    cp .env.example .env
    
    # Generate a secure JWT secret
    if command -v openssl &> /dev/null; then
        JWT_SECRET=$(openssl rand -hex 32)
        sed -i "s/your-secret-key-change-in-production-minimum-32-characters/$JWT_SECRET/" .env
        print_success ".env file created with secure JWT secret"
    else
        print_info ".env file created. Please update JWT_SECRET manually."
    fi
else
    print_info ".env file already exists, skipping..."
fi

# Start Docker services
print_info "Starting Docker services..."
docker-compose up -d

print_success "Docker services started"

# Wait for PostgreSQL to be ready
print_info "Waiting for PostgreSQL to be ready..."
sleep 5

MAX_TRIES=30
TRIES=0
until docker exec codesprint_db pg_isready -U codesprint > /dev/null 2>&1 || [ $TRIES -eq $MAX_TRIES ]; do
    echo -n "."
    sleep 1
    TRIES=$((TRIES+1))
done
echo ""

if [ $TRIES -eq $MAX_TRIES ]; then
    print_error "PostgreSQL failed to start"
    exit 1
fi

print_success "PostgreSQL is ready"

# Run database migrations
print_info "Running database schema..."
docker exec -i codesprint_db psql -U codesprint -d codesprint < database/schema.sql > /dev/null 2>&1 || print_info "Schema already exists (this is normal)"

print_info "Running database migrations..."
docker exec -i codesprint_db psql -U codesprint -d codesprint < database/migrations.sql > /dev/null 2>&1 || print_info "Migrations already applied (this is normal)"

print_success "Database initialized"

# Judge0 is called via RapidAPI, so there is no local judge0 container to wait for.
print_info "RapidAPI Judge0: ensure `RAPIDAPI_KEY` is configured in your backend environment"
print_success "Setup complete (backend can start immediately; Judge0 calls happen on demand)"

# Success message
echo ""
echo "============================================"
print_success "Setup Complete!"
echo "============================================"
echo ""
echo "Next steps:"
echo "1. Access the application at: http://localhost:8080"
echo "2. Create your account through the UI"
echo "3. Make yourself admin:"
echo ""
echo "   docker exec -it codesprint_db psql -U codesprint -d codesprint"
echo "   UPDATE users SET is_admin = TRUE, is_main_manager = TRUE WHERE email = 'your@email.com';"
echo "   \\q"
echo ""
echo "4. Refresh the page and you'll see the Admin button"
echo ""
echo "Useful commands:"
echo "  - View logs: docker-compose logs -f"
echo "  - Stop services: docker-compose down"
echo "  - Restart: docker-compose restart"
echo "  - Check status: docker-compose ps"
echo ""
