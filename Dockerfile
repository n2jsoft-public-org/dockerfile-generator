# syntax=docker/dockerfile:1.7
# Multi-stage build for dockerfile-gen: builds from source then produces an ultra-small (scratch) runtime image.

############################
# Build stage
############################
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS build
ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
ARG COMMIT=none
ARG DATE=unknown

WORKDIR /src

# Build deps (git for modules fetched via VCS)
RUN apk add --no-cache git

# Copy go module files first for better caching
COPY go.mod go.sum ./
# Download deps (with cache mount for speed between builds)
RUN --mount=type=cache,target=/go/pkg/mod go mod download

# Copy the remainder of the source
COPY . .

# Build the binary (cache build artifacts)
# -trimpath removes local paths, -s -w strips symbols, -buildid= makes builds more reproducible.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go build -mod=readonly -trimpath -ldflags="-s -w -buildid= -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}" -o /out/dockerfile-gen .

############################
# Runtime stage (scratch)
############################
FROM scratch AS runtime
ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
ARG COMMIT=none

# Labels (architecture + version metadata)
LABEL org.opencontainers.image.title="dockerfile-gen" \
      org.opencontainers.image.description="CLI to generate optimized Dockerfiles for projects" \
      org.opencontainers.image.vendor="n2jsoft" \
      org.opencontainers.image.source="https://github.com/${GITHUB_REPOSITORY}" \
      org.opencontainers.image.arch=$TARGETARCH \
      org.opencontainers.image.os=$TARGETOS \
      org.opencontainers.image.version=$VERSION \
      org.opencontainers.image.revision=$COMMIT

# Copy statically linked binary
COPY --from=build /out/dockerfile-gen /dockerfile-gen

# Non-root (65532 is a common nobody/nonroot uid:gid, e.g., distroless)
USER 65532:65532

# Workdir where user code can be mounted (declared as volume for clarity)
WORKDIR /src
VOLUME /src

ENTRYPOINT ["/dockerfile-gen", "--path", "/src"]
CMD ["--version"]

############################
# (Optional) Alpine runtime stage if you need a shell or CA certs
############################
# To use, build with: --target alpine-runtime
FROM alpine:3.22 AS alpine-runtime
ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
ARG COMMIT=none
LABEL org.opencontainers.image.title="dockerfile-gen" \
      org.opencontainers.image.description="CLI to generate optimized Dockerfiles for projects (alpine runtime)" \
      org.opencontainers.image.vendor="n2jsoft" \
      org.opencontainers.image.source="https://github.com/${GITHUB_REPOSITORY}" \
      org.opencontainers.image.arch=$TARGETARCH \
      org.opencontainers.image.os=$TARGETOS \
      org.opencontainers.image.version=$VERSION \
      org.opencontainers.image.revision=$COMMIT
RUN adduser -D -u 10001 app
COPY --from=build /out/dockerfile-gen /usr/local/bin/dockerfile-gen
USER app
WORKDIR /src
VOLUME /src
ENTRYPOINT ["dockerfile-gen", "--path", "/src"]
CMD ["--version"]
