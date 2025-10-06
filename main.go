package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/n2jsoft-public-org/dotnet-dockerfile-generator/internal/config"
	_ "github.com/n2jsoft-public-org/dotnet-dockerfile-generator/internal/dotnet" // register dotnet generator
	"github.com/n2jsoft-public-org/dotnet-dockerfile-generator/internal/generator"
	_ "github.com/n2jsoft-public-org/dotnet-dockerfile-generator/internal/golang" // register go generator
	"github.com/n2jsoft-public-org/dotnet-dockerfile-generator/internal/unidiff"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// translateLegacyLongFlags rewrites legacy single-dash long flags (-path) to the
// canonical double-dash form (--path) for backward compatibility with the prior
// implementation that accepted -path style long flags using the stdlib flag pkg.
func translateLegacyLongFlags(args []string) []string {
	legacy := map[string]struct{}{
		"-path":       {},
		"-language":   {},
		"-dockerfile": {},
		"-dry-run":    {},
		"-version":    {},
	}
	out := make([]string, 0, len(args))
	for _, a := range args {
		if _, ok := legacy[a]; ok {
			out = append(out, "-"+a) // prepend one more '-' -> '--path'
			continue
		}
		// handle assignment forms like -path=VALUE
		if eq := strings.IndexByte(a, '='); eq > 0 {
			prefix := a[:eq]
			if _, ok := legacy[prefix]; ok {
				out = append(out, "-"+a)
				continue
			}
		}
		out = append(out, a)
	}
	return out
}

func main() {
	// Backward compatibility: adjust os.Args so cobra can parse old single-dash long flags.
	if len(os.Args) > 1 {
		translated := translateLegacyLongFlags(os.Args[1:])
		os.Args = append([]string{os.Args[0]}, translated...)
	}

	var projectPath string
	var dockerfileName string
	var language string
	var dryRun bool
	var versionLower bool
	var versionUpper bool

	rootCmd := &cobra.Command{
		Use:   "dockerfile-gen",
		Short: "Generate optimized multi-stage Dockerfiles for .NET and Go projects",
		Long: `dockerfile-gen generates multi-stage Dockerfiles optimized for build caching.
It supports autodetection of project type (.csproj / go.mod) or explicit language selection.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Version first so -v doesn't require --path
			if versionLower || versionUpper {
				fmt.Printf("%s version %s (commit %s, built %s)\n", filepath.Base(os.Args[0]), version, commit, date)
				return nil
			}

			if projectPath == "" {
				return fmt.Errorf("project path is required (use -p / --path)")
			}

			if _, err := os.Stat(projectPath); os.IsNotExist(err) {
				return fmt.Errorf("project path not found: %s", projectPath)
			}

			rootPath := findRepositoryRoot(projectPath)
			if rootPath == "" {
				return fmt.Errorf("cannot find repository root")
			}

			projectDirectory := projectPath
			if fi, err := os.Stat(projectDirectory); err == nil && !fi.IsDir() {
				projectDirectory = filepath.Dir(projectDirectory)
			}

			cfgPath := filepath.Join(projectDirectory, config.DefaultDockerBuildFileName)
			cfg := config.Default()
			if data, err := os.Stat(cfgPath); err == nil && !data.IsDir() {
				loaded, err2 := config.Load(cfgPath)
				if err2 != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to load config: %v\n", err2)
				} else {
					cfg = loaded
				}
			}

			// If language flag not set, use config
			if language == "" && cfg.Language != "" {
				language = strings.ToLower(cfg.Language)
			}

			// Autodetect if still empty
			if language == "" {
				for _, g := range generator.All() {
					ok, _ := g.Detect(projectPath)
					if ok {
						language = g.Name()
						break
					}
				}
			}
			if language == "" {
				return fmt.Errorf("could not detect language; provide -l / --language")
			}

			gen, exists := generator.Get(strings.ToLower(language))
			if !exists {
				return fmt.Errorf("unsupported language '%s'", language)
			}

			project, additional, err := gen.Load(projectPath, rootPath)
			if err != nil {
				return fmt.Errorf("error loading project: %w", err)
			}

			dest := filepath.Join(projectDirectory, dockerfileName)

			if dryRun {
				tmp, err := os.CreateTemp("", "dockerfile-gen-*")
				if err != nil {
					return fmt.Errorf("error creating temp file: %w", err)
				}
				tmpPath := tmp.Name()
				_ = tmp.Close()
				defer os.Remove(tmpPath)
				if err := gen.GenerateDockerfile(project, additional, tmpPath, cfg); err != nil {
					return fmt.Errorf("error generating Dockerfile (dry-run): %w", err)
				}
				newBytes, err := os.ReadFile(tmpPath)
				if err != nil {
					return fmt.Errorf("error reading generated Dockerfile: %w", err)
				}
				var oldBytes []byte
				if _, err := os.Stat(dest); err == nil {
					oldBytes, _ = os.ReadFile(dest)
				}
				if string(oldBytes) == string(newBytes) {
					fmt.Printf("Dry run: no changes. %s is up to date.\n", dest)
					return nil
				}
				diff := unidiff.Unified(string(oldBytes), string(newBytes), dest)
				fmt.Println(diff)
				fmt.Println("Dry run: no file written.")
				return nil
			}

			if err := gen.GenerateDockerfile(project, additional, dest, cfg); err != nil {
				return fmt.Errorf("error generating Dockerfile: %w", err)
			}
			fmt.Printf("Successfully generated %s (%s) for project %s\n", dockerfileName, language, projectPath)
			return nil
		},
	}

	// Flags
	f := rootCmd.Flags()
	f.StringVarP(&projectPath, "path", "p", "", "Path to the project (directory, .csproj, or go.mod) (required)")
	f.StringVarP(&dockerfileName, "dockerfile", "f", "Dockerfile", "Name of the Dockerfile to generate")
	f.StringVarP(&language, "language", "l", "",
		"Language override (dotnet, go). If empty attempts autodetect or config")
	f.BoolVarP(&dryRun, "dry-run", "d", false, "Do not write file; show diff between existing and generated content")
	f.BoolVarP(&versionLower, "version", "v", false, "Print version information and exit")
	// Uppercase alias
	f.BoolVarP(&versionUpper, "Version", "V", false, "Print version information and exit")
	_ = f.MarkHidden("Version") // keep -V working but hide from help

	rootCmd.Example = `  dockerfile-gen -p ./src/WebApi/WebApi.csproj
  dockerfile-gen -p ./service -l go -f Dockerfile.service
  dockerfile-gen -p ./src/WebApi -d
  dockerfile-gen -v`

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
