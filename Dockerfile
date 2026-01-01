# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies (sqlite requires cgo)
RUN apk add --no-cache gcc musl-dev

WORKDIR /app

COPY go.mod ./
# COPY go.sum ./ 
# (Commented out go.sum until we actually run go mod tidy inside a container or locally)

RUN go mod download || echo "Skipping download as go.sum is missing"

COPY . .

# Build the binary
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/server

# Final stage
FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/main .

# Expose port
EXPOSE 8080

# Run
CMD ["./main"]
