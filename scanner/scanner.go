package scanner

import (
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/mergestat/kyc"
	"github.com/mergestat/kyc/scanner/files/docker"
)

type Scanner interface {
	// Supports returns true if the given file is supported, else it returns false.
	Supports(file *object.File) bool

	// Scan reads the content of the file and emit facts based on it.
	Scan(file *object.File) ([]*kyc.Fact, error)
}

func DockerScanner() Scanner { return &docker.DockerfileScanner{} }
