# ==============================================================================
# Build Stage
# ==============================================================================
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy module definition
COPY go.mod ./

# Copy source directory
COPY src/ ./src/

# Build statically-linked executable binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o colearning-agent ./src/cmd/api

# ==============================================================================
# Final Runtime Stage
# ==============================================================================
FROM alpine:3.20

# Install basic runtime utilities for security certificates and healthchecks
RUN apk add --no-cache ca-certificates tzdata wget

# Create non-root system user and group
RUN addgroup -g 10001 -S appgroup && \
    adduser -u 10001 -S appuser -G appgroup

WORKDIR /app

# Copy compiled executable from builder stage
COPY --from=builder /app/colearning-agent /app/colearning-agent

# Copy HTML & HTMX template files required by view.NewRenderer
COPY --from=builder /app/src/templates /app/src/templates

# Assign permissions
RUN chown -R appuser:appgroup /app

# Run as non-root user
USER appuser:appgroup

# Configuration defaults
ENV PORT=8080
EXPOSE 8080

# Container healthcheck targeting /healthz endpoint
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -q -O - http://localhost:${PORT}/healthz || exit 1

ENTRYPOINT ["/app/colearning-agent"]
