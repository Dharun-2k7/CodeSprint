#!/bin/bash

# Codesprint Setup Script
# This script automates the initial setup of the Codesprint platform locally
set -e

echo "============================================"
echo "  Codesprint Setup Script"
echo "============================================"
echo ""

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

print_success() { echo -e "${GREEN}✓ $1${NC}"; }
print_error() { echo -e "${RED}✗ $1${NC}"; }
print_info() { echo -e "${YELLOW}ℹ $1${NC}"; }

if [ ! -f .env ]; then
    print_info "Creating .env file..."
    cp .env.example .env
    
    if command -v openssl &> /dev/null; then
        JWT_SECRET=$(openssl rand -hex 32)
        sed -i "s/your_jwt_secret_here/$JWT_SECRET/" .env
        print_success ".env file created with secure JWT secret"
    else
        print_info ".env file created. Please update JWT_SECRET manually."
    fi
else
    print_info ".env file already exists, skipping..."
fi

print_info "Downloading Go Modules..."
go mod tidy

print_info "Building specific binaries..."
go build -o main

print_success "Build complete! Start the app using:"
echo "  ./main"
