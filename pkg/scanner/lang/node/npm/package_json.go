package npm

import (
	"context"
	"encoding/json"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/mergestat/kyc/pkg/scanner"
	"io"
	"strings"
)

type PackageJson struct {
	LockfileVersion string `json:"lockfileVersion"`

	Dependencies     map[string]string `json:"dependencies"`
	DevDependencies  map[string]string `json:"devDependencies"`
	PeerDependencies map[string]string `json:"peerDependencies"`
}

type PackageJsonScanner struct{}

func (p *PackageJsonScanner) Supports(file *object.File) bool {
	return file.Mode.IsFile() && strings.HasSuffix(file.Name, "package.json")
}

func (p *PackageJsonScanner) Scan(_ context.Context, file *object.File, _ ...string) (_ []scanner.Fact, err error) {
	var facts []scanner.Fact

	// read the file content to parse
	var reader io.ReadCloser
	if reader, err = file.Reader(); err != nil {
		return nil, err
	}
	defer reader.Close()

	var packageJson PackageJson
	if err = json.NewDecoder(reader).Decode(&packageJson); err != nil {
		return nil, err
	}

	// for each {dependency, devDependency, peerDependency}, emit a fact
	for name, version := range packageJson.Dependencies {
		var val = map[string]any{"name": name, "version": version}
		facts = append(facts, scanner.Fact{Key: "@node/npm/dependency", Value: val})
	}
	for name, version := range packageJson.DevDependencies {
		var val = map[string]any{"name": name, "version": version, "dev": true}
		facts = append(facts, scanner.Fact{Key: "@node/npm/dependency", Value: val})
	}
	for name, version := range packageJson.PeerDependencies {
		var val = map[string]any{"name": name, "version": version, "peer": true}
		facts = append(facts, scanner.Fact{Key: "@node/npm/dependency", Value: val})
	}

	return facts, nil
}

// register the PackageJsonScanner with scanner registry
func init() { scanner.Register("node/npm/package-json", &PackageJsonScanner{}) }
