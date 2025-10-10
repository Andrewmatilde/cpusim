#!/bin/bash

# Generate TypeScript types from OpenAPI specification
echo "Generating TypeScript types from OpenAPI specification..."

# Use npx to run swagger-codegen-cli
npx @openapitools/openapi-generator-cli generate \
  -i ../api/dashboard.openapi.yaml \
  -g typescript-fetch \
  -o ./src/api/generated \
  --additional-properties=supportsES6=true,npmVersion=10.0.0,typescriptThreePlus=true

echo "TypeScript types generated successfully!"