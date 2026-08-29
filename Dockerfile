# Build stage: compile the Go binary in an isolated Go environment.
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Copy dependency files first so Docker can reuse this layer when only source code changes.
COPY go.mod go.sum ./
RUN go mod download

# Copy the project source and compile a Linux binary for the final container.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /go_shope .

# Runtime stage: keep the final image small; it only needs the compiled binary and CA certificates.
FROM alpine:3.22

RUN apk add --no-cache ca-certificates
WORKDIR /app

COPY --from=builder /go_shope /app/go_shope
COPY config.yaml /app/config.yaml
# The Gin routes serve the buyer and merchant pages directly from this folder.
COPY web /app/web

EXPOSE 8080

CMD ["/app/go_shope"]
