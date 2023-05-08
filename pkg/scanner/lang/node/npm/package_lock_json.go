package npm

import (
	"context"
	"encoding/json"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/mergestat/kyc/pkg/scanner"
	"io"
	"strings"
)

type PackageLock struct {
	Name    string `json:"name"`
	Version string `json:"version"`

	Packages map[string]struct {
		Version   string `json:"version"`
		Resolved  string `json:"resolved"`
		Integrity string `json:"integrity"`
	} `json:"packages"`
}

type PackageLockScanner struct{}

func (p *PackageLockScanner) Supports(file *object.File) bool {
	return file.Mode.IsFile() && strings.HasSuffix(file.Name, "package-lock.json")
}

func (p *PackageLockScanner) Scan(_ context.Context, file *object.File, _ ...string) (_ []scanner.Fact, err error) {
	var facts []scanner.Fact

	// read the file content to parse
	var reader io.ReadCloser
	if reader, err = file.Reader(); err != nil {
		return nil, err
	}
	defer reader.Close()

	var packageLock PackageLock
	if err = json.NewDecoder(reader).Decode(&packageLock); err != nil {
		return nil, err
	}

	for name, pkg := range packageLock.Packages {
		var val = map[string]any{"path": name, "version": pkg.Version, "resolved": pkg.Resolved, "integrity": pkg.Integrity}
		facts = append(facts, scanner.Fact{Key: "@node/npm/dependency-locked", Value: val})
	}

	return facts, nil
}

// register the PackageLockScanner with scanner registry
func init() { scanner.Register("node/npm/package-lock", &PackageLockScanner{}) }
