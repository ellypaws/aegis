# Stage 1: Build Frontend
FROM oven/bun:latest AS frontend

WORKDIR /frontend

# Initialize git submodules (for discordgo or other submodules)
RUN git submodule update --init --recursive || true

# Copy frontend dependency files
COPY app/package.json ./
# Copy lockfile if it exists, otherwise this step is skipped/handled
COPY app/bun.lock* ./

# Install dependencies  
RUN bun install

# Copy frontend source
COPY app/ ./

# Build frontend
RUN bun run build

# Stage 2: Build Backend
FROM golang:alpine AS backend

WORKDIR /build

# Install git and build tools
RUN apk add --no-cache git build-base

# Copy go mod files
COPY go.mod go.sum ./

# Copy vendor files
COPY src ./src

# Download modules
RUN go mod download

# Copy source code
COPY . .

# Copy built frontend assets to the expected location (app/dist)
# The Go application expects 'app/dist' to be present for embedding
COPY --from=frontend /frontend/dist ./app/dist
COPY cmd/src/rsrc_windows_amd64.syso cmd/

# Build the binary
# -trimpath for reproducible builds
# -ldflags='-s -w' to strip debug info
# The Go version used is determined by the image (alpine latest) which satisfies go.mod
ARG BUILD_TAGS=""
RUN CGO_ENABLED=0 go build -trimpath -tags "${BUILD_TAGS}" -ldflags='-s -w' -o drigo ./cmd

# Stage 3: Final Runtime Image
FROM alpine:latest

WORKDIR /app
# Install runtime dependencies if needed (e.g. ca-certificates)
RUN apk add --no-cache ca-certificates

# Copy binary from backend builder
COPY --from=backend /build/drigo .

# Expose port (default 3000)
EXPOSE 3000

# Run the application
CMD ["./drigo"]
