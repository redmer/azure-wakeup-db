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
COPY --chmod=777 --from=builder /app/azure-wakeup-db /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/azure-wakeup-db"]
