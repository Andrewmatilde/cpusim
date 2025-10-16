#!/bin/bash
set -e

# Build frontend for production
echo "Building frontend..."
cd web
npm install
npm run build

# Create tarball
echo "Creating tarball..."
cd dist
tar -czf ../../dashboard-frontend.tar.gz .
cd ../..

echo "Frontend build complete: dashboard-frontend.tar.gz"
ls -lh dashboard-frontend.tar.gz
