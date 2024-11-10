# Use a minimal base image
FROM golang:1.23.2-bookworm as builder

# Get the proto dependencies
# WORKDIR /anyonefortennisproto/github.com/JamesBlackman-SLG/anyonefortennisproto
# COPY anyonefortennisproto/github.com/JamesBlackman-SLG/anyonefortennisproto ./
# RUN go mod download

# Set the working directory inside the container
WORKDIR /app

# Copy the Go server code from the source directory to the container
COPY consumer/ ./

# Build the Go server in release mode
RUN go clean -modcache
RUN go mod download
RUN go mod tidy
RUN go build -o group800consumer 
# Use a minimal runtime image for the final stage
FROM debian:bookworm-slim
# Install necessary libraries and CA certificates for TLS connections
RUN apt-get update && apt-get install -y --no-install-recommends \
  ca-certificates && \
  rm -rf /var/lib/apt/lists/*
# Set the working directory inside the runtime container
WORKDIR /root/

# Copy the built binary from the builder stage
COPY --from=builder /app/group800consumer .

# Set executable permissions for the Go binary
RUN chmod +x /root/group800consumer

# Copy the certificate files from the source folder to the runtime container
COPY ../.certs /root/.certs

# Set permissions for the certificates
RUN chmod -R 600 /root/.certs

# Expose the port that the Go server listens on
EXPOSE 8443
# Set environment variable for gRPC port
ENV GROUP_800_GRPC_PORT=50051
ENV GROUP_800_ENV=production
ENV GROUP_800_TURSO_URL=libsql://group800-jamesblackman-slg.turso.io
ENV GROUP_800_TURSO_TOKEN=eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJhIjoicnciLCJpYXQiOjE3MzA5MzI4NzAsImlkIjoiNzM3MmIwOTktYTA1Zi00Njc5LTgzNmItYWNjOGUzOTIxOTAwIn0.MGq_WxkfVco49nnyqfWGxZ0WNH-VDj9F6x1Qr6W8iznwRgkRiuQt7wJ0gV978sAa4l6M4n7f4fWq38jXotSqCg

# Command to run the Go server
ENTRYPOINT ["./group800consumer"]
