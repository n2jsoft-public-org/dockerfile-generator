# dockerfile-gen (dotnet + go)

A tiny Go-powered CLI that generates optimized multi-stage Dockerfiles for:
- .NET (C#) projects (from a `.csproj` file or directory containing one)
- Go modules (from a `go.mod` file or directory)

It analyzes the project (and referenced projects for .NET), discovers shared build context files (like `nuget.config`, `Directory.Build.props`, `Directory.Packages.props`), and emits a reproducible Dockerfile tuned for build caching. A lightweight YAML config (`.dockerbuild`) lets you override base images and add packages for build/runtime stages.

## What's New
- Multi-language architecture via pluggable generators (`internal/dotnet`, `internal/golang`).
- Autodetection of language based on path (`.csproj`, `go.mod`).
- `--language` flag to explicitly force a language (`dotnet`, `go`).
- Short flag aliases: `-p` (path), `-f` (dockerfile), `-l` (language), `-d` (dry-run), `-v`/`-V` (version).
- Cobra-based CLI (supports `--long` + `-short` flags). Legacy single-dash long forms like `-path` are still accepted for backward compatibility.
- Config file `.dockerbuild` with optional fields:
  ```yaml
  language: dotnet            # optional; overrides autodetection
  base:                       # runtime stage image & packages
    image: mcr.microsoft.com/dotnet/aspnet:9.0-alpine
    packages:
      - icu-data-full
      - icu-libs
  base-build:                 # build stage image & packages
    image: mcr.microsoft.com/dotnet/sdk:9.0-alpine
    packages:
      - git
  ```
  Example for Go:
  ```yaml
  language: go
  base:
    image: alpine:3.20
    packages:
      - ca-certificates
  base-build:
    image: golang:1.23-alpine
    packages:
      - build-base
  ```

If `language` is omitted in the config and `--language` flag is not provided, autodetection runs.

## Why?
Building Docker images often causes unnecessary cache invalidations because the entire source tree is copied before dependency restore. This tool:
- (dotnet) Copies only referenced project & context files before `dotnet restore` for better caching.
- (go) Uses multi-stage build with module and build caches mounted.
- Lets you tune base / build images and add OS packages without hand-editing every Dockerfile.

## Key Features
- Multi-language: dotnet + go (extensible design).
- Project graph traversal for .NET (follows `<ProjectReference>` recursively, detects circular refs).
- Automatic inclusion of shared .NET context files:
  - First `nuget.config` under the repo root
  - All `Directory.Build.props` and `Directory.Packages.props` in the directory chain of each project
- YAML config overrides for base images & packages per stage.
- Parameterized Dockerfiles with sensible, cache-friendly layering.

## Installation
With Go installed (1.21+ recommended):
```bash
go install github.com/maxime-charles_n2jsoft/dotnet-dockerfile-gen@latest
```

Or build locally:
```bash
git clone <repo-url>
cd dotnet-dockerfile-gen
go build -o dockerfile-gen ./...
```

## CLI Usage
General form:
```
dockerfile-gen --path <project-or-dir> [--language dotnet|go] [--dockerfile Dockerfile] [--dry-run]
```
(Short forms also accepted: `-p`, `-l`, `-f`, `-d`. Version: `-v` / `-V`.)
Legacy: single-dash long forms (`-path`, `-language`, etc.) still work but are deprecated.

### Flags
- `-p, --path` (string, required):
  - .NET: path to a `.csproj` file OR a directory containing exactly one `.csproj`.
  - Go: path to a `go.mod` file OR the module root directory.
- `-l, --language` (optional): Force generator (`dotnet`, `go`). Skips autodetection.
- `-f, --dockerfile` (optional): Output file name (default `Dockerfile`).
- `-d, --dry-run` (optional): Generate to a temp file and print a unified diff vs existing Dockerfile (no file written).
- `-v, -V, --version` (optional): Print version metadata.

### Exit Codes
- `0` success
- `1` validation or processing failure

## Examples
### Generate for a .NET web project
```bash
dockerfile-gen -p ./src/WebApi/WebApi.csproj
```
### Generate for a directory that contains one .csproj
```bash
dockerfile-gen --path ./src/WebApi
```
### Force language
```bash
dockerfile-gen -p ./src/WebApi --language dotnet
```
### Go module
```bash
dockerfile-gen -p ./service
```
### Go module specifying language & output name
```bash
dockerfile-gen -p ./service -l go -f Dockerfile.service
```

### With a config file
Place `.dockerbuild` next to your `.csproj` or `go.mod`:
```yaml
language: dotnet
base:
  image: mcr.microsoft.com/dotnet/aspnet:9.0-alpine
  packages:
    - icu-data-full
base-build:
  image: mcr.microsoft.com/dotnet/sdk:9.0-alpine
  packages:
    - git
```
Then run:
```bash
dockerfile-gen -p ./src/WebApi/WebApi.csproj
```

## Generated Dockerfile (Dotnet Overview)
Stages (simplified):
1. `base` runtime image (aspnet) + optional packages
2. `base_build` SDK image + optional packages
3. `build` copies project graph & context, runs `dotnet restore`, then source & `dotnet build`
4. `publish` runs `dotnet publish`
5. `final` runtime image with published output

Supported build args (dotnet):
- `TARGET_DOTNET_VERSION` (default `9.0`)
- `BUILD_CONFIGURATION` (default `Release`)
- `APP_VERSION` (default `0.0.1`)
- `NuGetPackageSourceToken_gh` (if private feed credentials pattern is used)

## Generated Dockerfile (Go Overview)
Stages:
1. `build` (golang:<version>-alpine or override) with module & build caches
2. `final` (alpine or override)

Build arg:
- `GO_VERSION` (default in template `1.23` unless overridden by base-build image)

## .NET Context Discovery
For each project (root + referenced):
- Walks upward to repo root capturing `Directory.Build.props` and `Directory.Packages.props`.
- Adds first discovered `nuget.config` once globally.
- Ensures unique file copy entries.

## Config File Reference (`.dockerbuild`)
```yaml
language: dotnet|go   # optional
base:
  image: <string>     # optional runtime stage base image
  packages:           # optional list of apk packages to install (alpine-based images)
    - pkg1
    - pkg2
base-build:
  image: <string>     # optional build stage base image
  packages:
    - build-pkg
```
Missing fields are ignored; no defaults are forced except `language` fallback to autodetect.

## Autodetection Logic
- If `--language` provided: use that.
- Else if config `language` set: use that.
- Else detect:
  - `.csproj` file path or directory containing exactly one `.csproj` -> `dotnet`
  - Directory or file containing `go.mod` -> `go`

## Version Flag
```bash
dockerfile-gen -v
```
Outputs:
```
<binary> version <semver> (commit <short>, built <date>)
```

## Troubleshooting
- Not detected: pass `-l` manually.
- Multiple `.csproj` in directory: specify a single file path.
- User permissions (dotnet): Provide `APP_UID` user in image or remove `USER $APP_UID` line.
- Private NuGet feeds: Provide `NuGetPackageSourceToken_gh` build arg; adapt template if feed name differs.

## Roadmap / Ideas
- Tests for generators.
- Support for .sln file as root.
- Language-specific configuration extensions.
- Automatic `.dockerignore` suggestion.
- Additional languages (Node.js, Python) via new generator packages.

## Contributing
1. Fork & clone.
2. Create a feature branch.
3. Update / add tests (future).
4. Submit PR.

## License
Add a `LICENSE` file (MIT / Apache-2.0 recommended).

## Disclaimer
Generated Dockerfiles are a starting point; review for security hardening before production use.
