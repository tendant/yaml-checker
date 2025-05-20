FROM node:18-alpine AS frontend-builder

WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm install
COPY frontend/ ./
RUN npm run build

FROM golang:1.20-alpine AS backend-builder

# Install build dependencies
RUN apk add --no-cache git

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY main.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -o yaml-checker .

FROM alpine:3.18

# Install yq
RUN apk add --no-cache wget bash && \
    wget https://github.com/mikefarah/yq/releases/latest/download/yq_linux_amd64 -O /usr/bin/yq && \
    chmod +x /usr/bin/yq

WORKDIR /app

# Copy the Go binary from the backend builder
COPY --from=backend-builder /app/yaml-checker .

# Copy the frontend build from the frontend builder
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist

# Expose the port the app runs on
EXPOSE 8082

# Set default environment variables
ENV PORT=8082 \
    HOST="" \
    REPO_OWNER="" \
    REPO_NAME="" \
    BRANCH="main" \
    GITHUB_TOKEN="" \
    FILE_PATHS="config.yaml,config/app.yaml,deploy/values.yaml"

# Command to run the application
CMD ["./yaml-checker"]
