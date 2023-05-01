package docker

import (
	"bytes"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/mergestat/kyc"
	"io"
)

type DockerfileScanner struct{}

func (d *DockerfileScanner) Supports(file *object.File) bool {
	return file.Mode.IsFile() && file.Name == "Dockerfile"
}

func (d *DockerfileScanner) Scan(file *object.File) (_ []*kyc.Fact, err error) {
	reader, _ := file.Reader()
	defer reader.Close()

	var content bytes.Buffer
	if _, err = io.Copy(&content, reader); err != nil {
		return nil, err
	}

	// scan the contents of the dockerfile and collect facts
	var facts []*kyc.Fact

	return facts, nil
}
