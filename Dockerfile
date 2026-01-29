# Build stage
FROM golang:1.25-alpine AS builder

# Set up build environment
ARG TARGETOS
ARG TARGETARCH
ARG VERSION
ARG COMMIT
ARG BUILD_DATE

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /build

# Copy go mod files first for better caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -a -installsuffix cgo \
    -ldflags="-w -s -X github.com/mondu-ai/gar-credential-provider/internal/version.Version=${VERSION} -X github.com/mondu-ai/gar-credential-provider/internal/version.GitCommit=${COMMIT} -X github.com/mondu-ai/gar-credential-provider/internal/version.BuildTime=${BUILD_DATE}" \
    -o gar-credential-provider \
    ./cmd/gar-credential-provider

# Runtime stage - using alpine for chroot (needed to restart kubelet in DaemonSet mode)
FROM alpine:3.23

# Add metadata labels
LABEL org.opencontainers.image.title="GAR Credential Provider"
LABEL org.opencontainers.image.description="Kubelet credential provider plugin for Google Artifact Registry using GCP Workload Identity Federation"
LABEL org.opencontainers.image.source="https://github.com/mondu-ai/gar-credential-provider"
LABEL org.opencontainers.image.documentation="https://github.com/mondu-ai/gar-credential-provider/blob/main/README.md"
LABEL org.opencontainers.image.licenses="MIT"

# Copy binary
COPY --from=builder /build/gar-credential-provider /gar-credential-provider

# Set entrypoint
ENTRYPOINT ["/gar-credential-provider"]
