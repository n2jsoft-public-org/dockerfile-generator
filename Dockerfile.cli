# syntax=docker/dockerfile:1.7
# Minimal container image for the dotnet-dockerfile-gen CLI
ARG TARGETOS
ARG TARGETARCH
FROM alpine:3.20

# Provide architecture metadata as labels (filled by buildx)
LABEL org.opencontainers.image.title="dotnet-dockerfile-gen" \
      org.opencontainers.image.description="CLI to generate optimized Dockerfiles for .NET projects" \
      org.opencontainers.image.vendor="n2jsoft" \
      org.opencontainers.image.source="https://github.com/${GITHUB_REPOSITORY}" \
      org.opencontainers.image.arch=$TARGETARCH \
      org.opencontainers.image.os=$TARGETOS

# Create non-root user
RUN adduser -D -u 10001 app

WORKDIR /app
# The binary will be injected by GoReleaser into the build context root.
COPY dotnet-dockerfile-gen /usr/local/bin/dotnet-dockerfile-gen

USER app
ENTRYPOINT ["dotnet-dockerfile-gen"]
CMD ["--version"]
