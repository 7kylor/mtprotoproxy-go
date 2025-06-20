# syntax=docker/dockerfile:1

# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install git for go modules
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY main.go ./

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o mtproxy .

# Final stage
FROM gcr.io/distroless/static:latest

# Copy the binary
COPY --from=builder /app/mtproxy /usr/local/bin/mtproxy

# Run as non-root user
USER 65534:65534

# Expose proxy port and metrics port
EXPOSE 443 8080

ENTRYPOINT ["/usr/local/bin/mtproxy"] 