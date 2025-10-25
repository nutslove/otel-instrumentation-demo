FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev sqlite-dev git

COPY go.mod ./
COPY main.go .
RUN go mod tidy
RUN go mod download

RUN CGO_ENABLED=1 go build -o go-service main.go

FROM alpine:latest

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache sqlite-libs ca-certificates

COPY --from=builder /app/go-service .

# Create data directory
RUN mkdir -p /data

EXPOSE 8080

CMD ["./go-service"]
