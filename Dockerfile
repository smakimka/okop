# Use the official Golang image
FROM golang:1.24 AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o parser cmd/parser/main.go

# Start a new stage from scratch
FROM alpine:latest

# Copy the Pre-built binary file and config from the previous stage
COPY --from=builder /app/parser .
COPY --from=builder /app/proxies.json .

# Command to run the executable
CMD ["./parser"]
