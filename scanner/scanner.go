package scanner

import (
	"github.com/go-git/go-git/v5/plumbing/object"
)

type Fact struct {
	Key   string
	Value any
}

type Scanner interface {
	// Name returns the full name of the scanner
	Name() string

	// Supports returns true if the given file is supported, else it returns false.
	Supports(file *object.File) bool

	// Scan reads the content of the file and emit facts based on it.
	Scan(file *object.File) ([]Fact, error)
}

// collection of all registered scanners
var scanners = make(map[Scanner]struct{})

// Register registers a new scanner in the global registry of scanners.
func Register(scanner Scanner) { scanners[scanner] = struct{}{} }

// All returns a list of all registered scanners
func All() []Scanner {
	var all = make([]Scanner, 0, len(scanners))
	for scanner := range scanners {
		all = append(all, scanner)
	}
	return all
}
