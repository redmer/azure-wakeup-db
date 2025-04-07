# Build stage
FROM golang:alpine AS builder

# Set working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN go build -o main .

# Run stage
FROM scratch

# Copy binary from builder
COPY --from=builder /app/main .

# Run the application
CMD ["./main"]
