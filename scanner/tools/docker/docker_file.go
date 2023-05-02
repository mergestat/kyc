package docker

import (
	"bytes"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/mergestat/kyc/scanner"
	"io"
)

type DockerfileScanner struct{}

func (d *DockerfileScanner) Name() string { return "docker/dockerfile" }

func (d *DockerfileScanner) Supports(file *object.File) bool {
	return file.Mode.IsFile() && file.Name == "Dockerfile"
}

func (d *DockerfileScanner) Scan(file *object.File) (_ []scanner.Fact, err error) {
	reader, _ := file.Reader()
	defer reader.Close()

	var content bytes.Buffer
	if _, err = io.Copy(&content, reader); err != nil {
		return nil, err
	}

	// scan the contents of the dockerfile and collect facts
	var val = map[string]any{"name": "golang", "version": 1.18}
	var facts = []scanner.Fact{
		{Key: "@docker/dockerfile/image", Value: val},
	}

	return facts, nil
}
