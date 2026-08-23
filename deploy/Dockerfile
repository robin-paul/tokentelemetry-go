# Stage 1: Build Astro Static Frontend
FROM node:20-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Stage 2: Compile Pure-Go Hub Binary
FROM golang:1.25-alpine AS backend-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend-builder /app/internal/web/dist ./internal/web/dist
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/tt-server ./cmd/tt-server

# Stage 3: Minimal Production Image
FROM alpine:3.20
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=backend-builder /bin/tt-server /usr/local/bin/tt-server
EXPOSE 8000
VOLUME ["/data"]
ENV TT_DB_PATH=/data/tokentelemetry.db
ENTRYPOINT ["/usr/local/bin/tt-server", "--db", "/data/tokentelemetry.db", "--port", "8000"]
