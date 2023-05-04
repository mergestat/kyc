package scanner

import (
	"context"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type Fact struct {
	Key   string
	Value any
}

type Scanner interface {
	// Supports returns true if the given file is supported, else it returns false.
	Supports(file *object.File) bool

	// Scan reads the content of the file and emit facts based on it.
	Scan(ctx context.Context, file *object.File) ([]Fact, error)
}

// collection of all registered scanners
var scanners = make(map[string]Scanner)

// Register registers a new scanner in the global registry of scanners.
func Register(name string, scanner Scanner) { scanners[name] = scanner }

// All returns a list of all registered scanners
func All() map[string]Scanner { return scanners }
