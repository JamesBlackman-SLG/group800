#!/bin/bash

# Templ templates
templ generate

# Build and run the Go application
echo "Starting Go application..."
go run ./cmd
