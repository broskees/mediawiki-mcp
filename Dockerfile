FROM golang:1.21-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o mediawiki-mcp .

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /build/mediawiki-mcp .

EXPOSE 8080

CMD ["./mediawiki-mcp"]
