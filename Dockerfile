# ==========================================
# SopAndGo
# Architecture: Single Container (Caddy + Node + Go)
#
# RECOMMENDED: Run with docker-compose.yml + .env (see README and docs/ops/deployment.md):
#   docker compose up -d --build
# Set ORIGIN and other variables in .env (from .env.example).
# ==========================================

# --- Stage 1: Build Frontend (SvelteKit) ---
FROM node:22-slim AS frontend-builder
ARG APP_VERSION=dev
ENV APP_VERSION=${APP_VERSION}
WORKDIR /app/frontend

# We define the BACKEND_URL here during the build phase.
# Since the frontend and backend live in the same container, 
# '127.0.0.1' is the robust internal loopback address.
ENV BACKEND_URL=http://127.0.0.1:8080

COPY frontend/package*.json ./
RUN npm install
COPY frontend/ .
RUN npm run build

# --- Stage 2: Build Backend (Go) ---
FROM golang:1.26-alpine AS backend-builder
WORKDIR /app/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ .

# Build the main application and the data seeder
RUN go build -o /sopandgo ./cmd/sopandgo/main.go
RUN go build -o /seeder ./cmd/seed-demo-data/main.go

# --- Stage 3: Final Runtime Image ---
FROM node:22-slim
ARG APP_VERSION=dev
ENV APP_VERSION=${APP_VERSION}
WORKDIR /app

# 1. Install Caddy (Reverse Proxy)
# We use the official instructions to add the Caddy repository and install it.
# Caddy acts as the entry point, handling compression and routing internal traffic.
RUN apt-get update && apt-get install -y curl debian-keyring debian-archive-keyring apt-transport-https \
    && curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg \
    && curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | tee /etc/apt/sources.list.d/caddy-stable.list \
    && apt-get update && apt-get install -y caddy \
    && rm -rf /var/lib/apt/lists/*

# 2. Copy Build Artifacts
# Backend binaries
COPY --from=backend-builder /sopandgo ./sopandgo
COPY --from=backend-builder /seeder ./seeder
# Demo data
COPY backend/demo ./demo
# Frontend build
COPY --from=frontend-builder /app/frontend/build ./frontend-build
COPY --from=frontend-builder /app/frontend/package.json ./package.json
COPY --from=frontend-builder /app/frontend/package-lock.json ./package-lock.json

# Install only production dependencies for the Node server.
# The lockfile is required: npm 10 on node:22-slim exits 1 with
# "Cannot read properties of null (reading 'edgesOut')" when this
# install is resolved from package.json alone.
RUN npm ci --omit=dev

# 3. Caddy Configuration
COPY Caddyfile /etc/caddy/Caddyfile

# 4. Entrypoint Script (Process Supervisor)
# Normalizes ORIGIN (see script). Caddy uses 'exec' so it receives SIGTERM/SIGINT.
COPY docker-entrypoint.sh /app/entrypoint.sh
# Windows checkouts may use CRLF; kernel then looks for /bin/sh\r ("no such file or directory").
RUN sed -i 's/\r$//' /app/entrypoint.sh && chmod +x /app/entrypoint.sh

# --- Environment Configuration ---

# Persistence
ENV DATA_DIR=/app/data

# Internal Networking
# Node talks to Go via localhost inside the container
ENV BACKEND_URL=http://127.0.0.1:8080

# SvelteKit Proxy Headers
# Necessary for SvelteKit to trust the request origin when behind Caddy
ENV HOST_HEADER=x-forwarded-host
ENV PROTOCOL_HEADER=x-forwarded-proto

# Default Origin for the Go API (invite/reset links). Override in compose for production/NAS.
# SvelteKit 3 derives request origin from HOST_HEADER / PROTOCOL_HEADER above, not ORIGIN.
ENV ORIGIN=http://localhost:8087

# Body Size Limit
# Default is 512kb. Increased to ~50MB to support larger payloads.
ENV BODY_SIZE_LIMIT=52428800

EXPOSE 80
ENTRYPOINT ["/app/entrypoint.sh"]