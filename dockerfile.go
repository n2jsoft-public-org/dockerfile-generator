package main

import (
	_ "embed"
	"os"
	"text/template"
)

//go:embed dockerfile.tmpl
var dockerfileTemplate string

type DockerfileTemplateContext struct {
	AdditionalFilePaths []AdditionalFilePath
	Project             Project
}

func generateDockerfile(project Project, additionalFilePaths []AdditionalFilePath, destinationPath string) error {
	tmpl, err := template.New("dockerfile").Parse(dockerfileTemplate)
	if err != nil {
		return err
	}

	file, err := os.Create(destinationPath)
	if err != nil {
		return err
	}
	defer file.Close()

	return tmpl.Execute(file, DockerfileTemplateContext{
		AdditionalFilePaths: additionalFilePaths,
		Project:             project,
	})

}
