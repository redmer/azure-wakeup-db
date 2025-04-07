# Build stage
FROM golang:alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN go build -o azure-wakeup-db ./src

# Run stage
FROM scratch
COPY --from=builder /app/azure-wakeup-db .

CMD ["./azure-wakeup-db"]
