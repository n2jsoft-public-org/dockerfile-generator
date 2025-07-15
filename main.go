package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	var csprojPath string
	var dockerfileName string

	flag.StringVar(&csprojPath, "csproj", "", "Path to the .csproj file (required)")
	flag.StringVar(&dockerfileName, "dockerfile", "Dockerfile", "Name of the Dockerfile to generate (optional)")
	flag.Parse()

	if csprojPath == "" {
		fmt.Fprintf(os.Stderr, "Error: .csproj path is required\n")
		flag.Usage()
		os.Exit(1)
	}

	if _, err := os.Stat(csprojPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: .csproj file not found at path: %s\n", csprojPath)
		os.Exit(1)
	}

	if !strings.HasSuffix(strings.ToLower(csprojPath), ".csproj") {
		fmt.Fprintf(os.Stderr, "Error: File must have .csproj extension\n")
		os.Exit(1)
	}

	rootPath := findRepositoryRoot(csprojPath)
	if rootPath == "" {
		fmt.Fprintf(os.Stderr, "Error: Cannot find repository root\n")
		os.Exit(1)
	}

	project, err := loadProject(csprojPath, rootPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing .csproj file: %v\n", err)
		os.Exit(1)
	}

	additionalFilePaths, err := loadProjectContextFromProject(project, rootPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading project context: %v\n", err)
		os.Exit(1)
	}

	// Generate Dockerfile
	//	dockerfile := generateDockerfile(project, filepath.Base(csprojPath))

	// Write Dockerfile
	//	err = writeDockerfile(dockerfileName, dockerfile)
	//	if err != nil {
	//		fmt.Fprintf(os.Stderr, "Error writing Dockerfile: %v\n", err)
	//		os.Exit(1)
	//	}

	destinationPath := filepath.Join(filepath.Dir(csprojPath), dockerfileName)
	if err := generateDockerfile(project, additionalFilePaths, destinationPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating Dockerfile: %v\n", err)
	}
	fmt.Printf("Successfully generated %s for project %s\n", dockerfileName, csprojPath)
}
