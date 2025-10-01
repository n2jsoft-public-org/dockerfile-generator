# dotnet-dockerfile-gen

A tiny Go-powered CLI that generates an optimized multi-stage Dockerfile for a .NET (C#) application starting from a single `.csproj` file. It analyzes the project graph (project references), discovers shared build context files (like `nuget.config`, `Directory.Build.props`, `Directory.Packages.props`), and emits a reproducible Dockerfile that speeds up `dotnet restore` by copying only what is necessary early in the image build.

## Why?
Building Docker images for multi-project .NET solutions often causes unnecessary cache invalidations because the entire source tree is copied before `dotnet restore`. This tool:
- Copies only referenced project files first so `dotnet restore` is cache-friendly.
- Includes shared dependency/config files automatically.
- Leaves you with a clean, parameterized Dockerfile you can commit or regenerate.

## Key Features
- Project graph traversal (follows `<ProjectReference>` recursively).
- Circular reference detection (fails fast if a loop is found).
- Automatic inclusion of:
  - `nuget.config` (first one found under repo root)  
  - Any `Directory.Build.props` in the directory chain of each project  
  - Any `Directory.Packages.props` in the directory chain of each project
- Generates a multi-stage Alpine-based Dockerfile (runtime + build + publish stages).
- Parameterized with build args: `TARGET_DOTNET_VERSION`, `BUILD_CONFIGURATION`, `APP_VERSION`, `NuGetPackageSourceToken_gh`.
- Keeps copy layers minimal before `dotnet restore` for better caching.

## How It Works (High-Level Flow)
1. You pass a path to a `.csproj` file.
2. The tool walks upward from that path to locate the repository root (first directory containing a `.git` folder).
3. It parses the project file and recursively loads referenced projects.
4. It builds a unique list of all projects plus additional context files.
5. It renders `dockerfile.tmpl` with the gathered metadata to the desired Dockerfile path.

## Generated Dockerfile Structure
Stages:
1. `base` – Runtime image (`aspnet:<TARGET_DOTNET_VERSION>-alpine`), installs ICU globalization bits, sets UTF-8 locale, switches to `USER $APP_UID` (you must provide this user at build or runtime; see Notes).
2. `base_build` – SDK image with environment prepared for private NuGet feeds.
3. `build` – Copies all project + context files individually, runs `dotnet restore`, then copies the remainder of the source tree and compiles (`dotnet build`).
4. `publish` – Runs `dotnet publish` (self-contained toggle disabled with `/p:UseAppHost=false`).
5. `final` – Copies published output into the runtime image and sets `ENTRYPOINT` to the main DLL.

Important template details:
- Default `ARG TARGET_DOTNET_VERSION=9.0` controls both runtime + SDK images and the target framework (`net${TARGET_DOTNET_VERSION}`).
- Additional build args:
  - `BUILD_CONFIGURATION` (default `Release`)
  - `APP_VERSION` (default `0.0.1`, passed as `/p:Version`)
  - `NuGetPackageSourceToken_gh` (used to build `NuGetPackageSourceCredentials_gh` env var for authenticated feeds)
- Uses `COPY ["path/to/project.csproj", "path/to/"]` style to optimize layer invalidation.
- Uses `--chown=$APP_UID:$APP_UID` on the copy from `publish` to final stage.

## Requirements & Assumptions
- You have a Git repository (used to determine the root). If no `.git` directory is found upward from the `.csproj`, generation fails.
- The `.csproj` path you pass exists and is a valid XML project file.
- If you plan to use a non-root user, ensure the user with UID `$APP_UID` exists in the final image (the template does not create it). You may need to extend the Dockerfile or add a stage to add the user.
- The target framework of all referenced projects must match or be compatible with the `TARGET_DOTNET_VERSION` you build with.

## Installation
With Go installed (1.21+ recommended):

```bash
go install github.com/your-org-or-user/dotnet-dockerfile-gen@latest
```

(Replace the module path above with the actual repository path if different.)

Or build locally:

```bash
git clone <repo-url>
cd dotnet-dockerfile-gen
go build -o dotnet-dockerfile-gen ./...
```

## CLI Usage
```
dotnet-dockerfile-gen --csproj /absolute/or/relative/path/to/MyApp.csproj [--dockerfile Dockerfile.Custom]
```

### Flags
- `--csproj` (string, required): Path to the main project file you want to containerize.
- `--dockerfile` (string, optional): Output Dockerfile name. Default: `Dockerfile` (written alongside the `.csproj`).

### Exit Codes
- `0` success
- `1` validation or IO/parsing error (missing file, not a `.csproj`, cannot find git root, parse failure, circular reference, etc.)

## Example
Assume structure:
```
repo/
  .git/
  src/
    WebApi/WebApi.csproj
    Core/Core.csproj
    Shared/Shared.csproj
  nuget.config
  Directory.Build.props
```

Run:
```bash
dotnet-dockerfile-gen --csproj ./src/WebApi/WebApi.csproj
```
Produces `./src/WebApi/Dockerfile` with COPY entries for:
- `src/WebApi/WebApi.csproj`
- `src/Core/Core.csproj`
- `src/Shared/Shared.csproj`
- `nuget.config`
- `Directory.Build.props`

Then you can build:
```bash
docker build \
  --build-arg TARGET_DOTNET_VERSION=9.0 \
  --build-arg BUILD_CONFIGURATION=Release \
  --build-arg APP_VERSION=1.2.3 \
  --build-arg NuGetPackageSourceToken_gh=$NUGET_TOKEN \
  -t myapp:1.2.3 \
  -f src/WebApi/Dockerfile .
```

Run:
```bash
docker run -e ASPNETCORE_URLS=http://+:8080 -p 8080:8080 myapp:1.2.3
```

## Working With Private NuGet Feeds
If `nuget.config` defines a source named `gh`, the template exposes:
- Build arg: `NuGetPackageSourceToken_gh`
- Env var inside build stage: `NuGetPackageSourceCredentials_gh` with a fixed username `docker_n2jsoft` and the supplied token as password.
Adjust the template if your feed name or credential format differs.

## Additional Files Discovery Logic
For each project (root + referenced):
- Scans upward (toward repo root) collecting `Directory.Build.props` and `Directory.Packages.props` found at each directory level.
- Adds the first `nuget.config` found (only once globally) anywhere under the repo root.
- Ensures uniqueness (no duplicates copied twice).

## Error Handling
Potential failures:
- Missing `.csproj` file or wrong extension.
- Cannot locate Git root (no `.git` found up the directory tree).
- XML parse error in project files.
- Circular project reference chain (detected and reported).
- Referenced project file missing (logged as warning and skipped unless it is the main project).

## Customizing the Dockerfile
You can safely edit the generated file after creation. To regenerate, just delete/rename it and re-run the tool. If you want permanent template changes, modify `dockerfile.tmpl` in the source and rebuild the CLI.

## Container Image
A published container image is built on tagged releases and pushed to GitHub Container Registry (GHCR).

Image references (examples for version v0.2.0):
```
# Exact version
ghcr.io/${GITHUB_REPOSITORY}:v0.2.0
# Major+minor tag
ghcr.io/${GITHUB_REPOSITORY}:v0.2
# Major tag
ghcr.io/${GITHUB_REPOSITORY}:v0
# Latest tag
ghcr.io/${GITHUB_REPOSITORY}:latest
```

Run the CLI via container:
```bash
docker run --rm ghcr.io/${GITHUB_REPOSITORY}:latest --version

docker run --rm -v "$PWD":"/workspace" -w /workspace \
  ghcr.io/${GITHUB_REPOSITORY}:latest \
  --csproj path/to/Project.csproj
```

Multi-arch support: linux/amd64, linux/arm64.

The image is based on `alpine` and runs as an unprivileged user (UID 10001).

## Releases
This repository is configured with GoReleaser and a GitHub Actions workflow to build and publish multi-platform archives automatically when you push a tag starting with `v`.

### Version Flag
The binary supports:
```bash
dotnet-dockerfile-gen --version
```
Output format:
```
<binary> version <semver> (commit <short-sha>, built <date>)
```
Values are injected at build time via `-ldflags` by GoReleaser (`main.version`, `main.commit`, `main.date`).

### Release Flow
1. Ensure `main` (or your release branch) is clean & tested.
2. Decide on a semantic version (e.g. `v0.1.0`).
3. Create and push the tag:
   ```bash
   git tag v0.1.0
   git push origin v0.1.0
   ```
4. GitHub Actions workflow `.github/workflows/release.yml` triggers:
   - Runs `go build` sanity check.
   - Executes GoReleaser to build archives for: `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`, `windows/amd64`, `windows/arm64`.
   - Attaches artifacts + checksums to the GitHub Release.

### Snapshot (Local) Release
You can simulate a release locally without publishing:
```bash
go install github.com/goreleaser/goreleaser@latest
goreleaser release --snapshot --clean
```
Artifacts will appear under `dist/`.

### Customizing
Edit `.goreleaser.yaml` to adjust:
- `goos` / `goarch` matrix
- Archive naming / included files
- Changelog filtering

If you add a `LICENSE` file it will automatically be bundled when present.

## Roadmap / Ideas
- Support solution (.sln) input.
- Optional pruning of unused files before final copy.
- Multi-arch build examples (BuildKit / `docker buildx`).
- Auto-detection of runtime vs console app (`OutputType`).
- User creation logic directly in template for non-root scenarios.
- Option to emit `.dockerignore` suggestions.

## Troubleshooting
- Layers not caching? Confirm that only project + context files changed; unrelated source edits should not invalidate earlier restore layers.
- Build fails with user error: Provide `--build-arg APP_UID=...` and ensure user exists (extend Dockerfile or remove `USER $APP_UID` if not needed).
- Wrong framework: Pass a matching `TARGET_DOTNET_VERSION` build arg (e.g., `8.0` for `net8.0`). Make sure projects target that framework.

## Contributing
1. Fork & clone.
2. Create a feature branch.
3. Add/update tests (if/when test suite exists).
4. Submit PR.

## License
(Repository does not yet contain a LICENSE file. Add one—MIT, Apache 2.0, etc.—to clarify usage.)

## Disclaimer
Use at your own risk. Generated Dockerfiles are a starting point; review security, vulnerability scanning, and production hardening practices before deploying.
