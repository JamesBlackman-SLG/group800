#!/bin/bash
npm run build:css

# Templ templates
templ generate

# Build and run the Go application
echo "Building Go application..."
go build -o group800web ./cmd
