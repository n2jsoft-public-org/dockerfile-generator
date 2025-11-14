# 🚀 dockerfile-gen (dotnet + go)

> Generate smart, cache-friendly multi-stage Dockerfiles for .NET & Go projects — instantly. ✨

<p align="center">
  <img alt="dockerfile-gen" src="https://img.shields.io/badge/dockerfile--gen-multi--language-blue?logo=docker"> 
  <img alt="Go Version" src="https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go"> 
  <img alt="License" src="https://img.shields.io/badge/License-MIT-green"> 
  <img alt="PRs Welcome" src="https://img.shields.io/badge/PRs-welcome-brightgreen"> 
</p>

---

## 📚 Table of Contents

- [Why?](#-why)
- [Features](#-key-features)
- [Installation](#-installation)
- [CLI Usage](#-cli-usage)
- [Examples](#-examples)
- [Config File](#-config-file-reference-dockerbuild)
- [Generated Dockerfile (Dotnet)](#-generated-dockerfile-dotnet-overview)
- [Generated Dockerfile (Go)](#-generated-dockerfile-go-overview)
- [.NET Context Discovery](#-net-context-discovery)
- [Autodetection Logic](#-autodetection-logic)
- [Version Output](#-version-flag)
- [Troubleshooting](#-troubleshooting)
- [Development](#-development)
- [Roadmap](#-roadmap--ideas)
- [Contributing](#-contributing)
- [License](#-license)
- [Disclaimer](#-disclaimer)

---

## 💡 Why?
Building Docker images often wastes time by copying the full source tree before dependency restore — destroying layer cache efficiency. This tool fixes that:

- 🧠 **Smart dependency staging**: Only copies the minimal project graph & shared context before `restore`.
- ⚡ **Better build caching**: Faster iterative builds locally & in CI.
- 🧩 **Multi-language support**: .NET & Go today (extensible design for more).
- 🛠 **Configurable**: Override base images & inject OS packages without editing Dockerfiles.
- 🔁 **Reproducible**: Deterministic layering strategy.

---

## 🔑 Key Features
- 🌐 Multi-language generators (`dotnet`, `go`).
- 🕵️ Autodetect project language (or force via `--language`).
- 🧬 Recursive .NET project graph traversal (follows `<ProjectReference>`; detects cycles).
- 📦 Automatic inclusion of shared files: `nuget.config`, `Directory.Build.props`, `Directory.Packages.props`.
- 🧾 YAML config (`.dockerbuild`) to override base/build images + `apk` package install lists.
- 🧪 Dry-run mode with unified diff output.
- 🐞 Optional verbose diagnostic logging (`--verbose`) showing detection & generation decisions (logs to stderr, leaving stdout clean).
- 🪄 Cache-friendly layering for both ecosystems.
- 🧱 Go builds use mount caches for modules & build output.

---

## 📥 Installation
### Option 1: Install via Go tooling (latest published version)
```bash
go install github.com/n2jsoft-public-org/dockerfile-generator@latest
```
> Module path (per `go.mod`): `github.com/n2jsoft-public-org/dockerfile-generator`

### Option 2: Clone + Makefile install (local build)
By default `make install` places the binary in the first entry of your `GOPATH`, i.e. `$GOPATH/bin` (commonly `$HOME/go/bin`). Ensure that directory is on your PATH.
```bash
git clone https://github.com/n2jsoft-public-org/dockerfile-generator.git
cd dockerfile-generator
make install                 # installs to $(go env GOPATH)/bin/dockerfile-gen
```
Override destination prefix (installs into `<prefix>/bin`):
```bash
make install PREFIX=$HOME/.local   # installs to $HOME/.local/bin/dockerfile-gen
```
Uninstall (matches the prefix you used to install):
```bash
make uninstall                     # removes $(go env GOPATH)/bin/dockerfile-gen
make uninstall PREFIX=$HOME/.local # removes $HOME/.local/bin/dockerfile-gen
```

### Windows notes
- The produced binary name is `dockerfile-gen.exe`.
- Default install path: `%GOPATH%\bin` (e.g. `C:\Users\<you>\go\bin`).
- Make sure that path is added to your `Path` environment variable.
- PowerShell example:
```powershell
$env:PATH += ";$HOME\go\bin"
```

### Option 3: Manual build
```bash
go build -o dockerfile-gen .
install -m 0755 dockerfile-gen "$HOME/go/bin/dockerfile-gen"   # or another directory on PATH
```
(Use `dockerfile-gen.exe` on Windows and place it in `%GOPATH%\bin`.)

---

## 🧪 CLI Usage
General form:
```
dockerfile-gen [--path <project-or-dir>] [--language dotnet|go] [--dockerfile Dockerfile] [--dry-run] [--verbose]
```
If `--path` is omitted it defaults to the current directory (`.`).
Short flags: `-p`, `-l`, `-f`, `-d`. Version: `-v` / `-V`.
Legacy (deprecated): single-dash long forms (`-path`, `-language`, ...).

### Flags
- `-p, --path` (string, optional, default `.`):
  - .NET: path to a `.csproj` OR a directory containing exactly one `.csproj`.
  - Go: path to a `go.mod` OR its module root directory.
- `-l, --language` (optional): Force generator (`dotnet`, `go`). If omitted, order: flag → config → autodetect.
- `-f, --dockerfile` (optional): Output file name (default `Dockerfile`).
- `-d, --dry-run` (optional): Generate to temp & print unified diff vs existing file (no write).
- `-v, -V, --version` (optional): Print version metadata.
- `--verbose` (optional): Enable debug logging (prints detection, config, and output path decisions to stderr; safe for piping stdout to files or other tools).

### Exit Codes
- `0` ✅ success
- `1` ❌ validation or processing failure

---

## 🧾 Examples
### Default (current directory)
```bash
dockerfile-gen
```
### .NET web project
```bash
dockerfile-gen -p ./src/WebApi/WebApi.csproj
```
### Directory containing exactly one `.csproj`
```bash
dockerfile-gen --path ./src/WebApi
```
### Force language
```bash
dockerfile-gen -p ./src/WebApi --language dotnet
```
### Go module (auto)
```bash
dockerfile-gen -p ./service
```
### Go module with output name
```bash
dockerfile-gen -p ./service -l go -f Dockerfile.service
```
### Verbose diagnostics (stderr logging)
```bash
dockerfile-gen -p ./service --verbose
```

### With a config file
Place `.dockerbuild` next to your `.csproj` or `go.mod`:
```yaml
language: dotnet
dotnet:
  sdk-version: "8.0"
base:
  image: mcr.microsoft.com/dotnet/aspnet:9.0-alpine
  packages:
    - icu-data-full
base-build:
  image: mcr.microsoft.com/dotnet/sdk:9.0-alpine
  packages:
    - git
```
Then:
```bash
dockerfile-gen -p ./src/WebApi/WebApi.csproj
```

---

## ⚙️ Config File Reference (`.dockerbuild`)
```yaml
language: dotnet|go         # optional
dotnet:                     # dotnet-specific config (optional)
  sdk-version: "9.0"        # target .NET version (default: "9.0")
base:
  image: <string>           # runtime stage base image
  packages:                 # apk packages (alpine-based images)
    - pkg1
    - pkg2
base-build:
  image: <string>           # build stage base image
  packages:
    - build-pkg
```
Missing fields are ignored. `language` falls back to autodetect.

Go example:
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

---

## 🛠 Generated Dockerfile (Dotnet Overview)
Stages (simplified):
1. `base` – runtime image (aspnet) + optional packages
2. `base_build` – SDK image + optional packages
3. `build` – copy project graph & context, `dotnet restore`, then copy source & `dotnet build`
4. `publish` – `dotnet publish`
5. `final` – runtime image with published output

Supported build args:
- `TARGET_DOTNET_VERSION` (default from config `dotnet.sdk-version` or `9.0` if not specified)
- `BUILD_CONFIGURATION` (default `Release`)
- `APP_VERSION` (default `0.0.1`)
- `NuGetPackageSourceToken_gh` (optional for private feed token injection)

To customize the target .NET version, set `dotnet.sdk-version` in your `.dockerbuild` file.

---

## 🛠 Generated Dockerfile (Go Overview)
Stages:
1. `build` – (golang:<version>-alpine or override) with module & build caches
2. `final` – (alpine or override)

Build arg:
- `GO_VERSION` (defaults in template to `1.23` unless overridden via base-build image)

---

## 🗂 .NET Context Discovery
Per project (root + referenced):
- Walk upward to repo root adding `Directory.Build.props` & `Directory.Packages.props`.
- Add first discovered `nuget.config` once globally.
- Ensure unique copy entries (no duplicates).

---

## 🔍 Autodetection Logic
Order of precedence:
1. `--language` flag (if provided)
2. Config `language` in `.dockerbuild`
3. Heuristics (on `--path` or `.` default directory):
   - Path to `.csproj` or directory with exactly one `.csproj` → `dotnet`
   - Directory/file containing `go.mod` → `go`

If both are present, whichever generator was registered first and detects successfully "wins" (current order: dotnet then go).

---

## 🧾 Version Flag
```bash
dockerfile-gen -v
```
Outputs:
```
<binary> version <semver> (commit <short>, built <date>)
```

---

## 🛟 Troubleshooting
| Issue | Tip |
|-------|-----|
| Not detected | Pass `-l` explicitly. |
| Multiple `.csproj` in directory | Specify a single file path. |
| Permissions / user mismatch | Provide `APP_UID` in build args or remove `USER $APP_UID` line after generation. |
| Private NuGet feeds | Provide `NuGetPackageSourceToken_gh` build arg; adapt template if feed name differs. |
| Need more insight into what the tool is doing | Re-run with `--verbose` to see detection & config decisions. |

---

## 🛠 Development
A `Makefile` is provided to streamline local workflows (linting, testing, releasing, Docker build, installation).

Common targets:

| Target | Description |
|--------|-------------|
| `make lint` | Run `golangci-lint` (auto-installs pinned version into `./bin`). |
| `make test` | Run tests with race detector + coverage profile. |
| `make coverage` | Run tests (if needed) and enforce coverage threshold (`COVERAGE_MIN`, default 60%). |
| `make build` | Build the CLI binary into `./bin/dockerfile-gen`. |
| `make install` | Build (if needed) and install to `${PREFIX}/bin` (default first GOPATH entry, e.g. `~/go/bin`). |
| `make uninstall` | Remove previously installed binary from `${PREFIX}/bin`. |
| `make snapshot` | Run GoReleaser in snapshot mode (no publish). |
| `make release` | Full GoReleaser release (requires tag + `GITHUB_TOKEN`). |
| `make docker` | Build a local Docker image `dockerfile-gen:dev`. |
| `make tidy` | Ensure `go.mod` / `go.sum` are tidy. |
| `make ci` | Run `lint` then `test` (approximate CI pipeline). |
| `make clean` | Remove build artifacts (`bin/`, `dist/`, coverage file). |

Environment overrides:
- `GOLANGCI_LINT_VERSION` to pin a different golangci-lint version.
- `COVERAGE_MIN` to raise/lower the required coverage percentage.
- `VERSION` for custom build version (defaults to `dev`).
- `PREFIX` install destination root (default first GOPATH entry).

Example:
```bash
make lint
make install                         # installs into $(go env GOPATH)/bin
make install PREFIX=$HOME/.local     # alt location
make uninstall PREFIX=$HOME/.local   # remove if needed
```
Ensure `$(go env GOPATH)/bin` or `PREFIX/bin` is on your PATH.

> Tip: Run `make help` to list available targets.

---

## 🤝 Contributing
1. Fork & clone
2. Create a feature branch
3. Add / update tests (future harness)
4. Open a PR 🚀

---

## 📄 License
Planned: MIT (add `LICENSE` file). Feel free to use & adapt — but confirm license once added.

---

## ⚠️ Disclaimer
Generated Dockerfiles are a strong starting point — always review for security hardening (non-root users, pinned versions, SBOM, vuln scanning) before production deployment.

---

Made with ❤️ for fast, reliable container builds.
