package kyc

import (
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/mergestat/kyc/scanner"
	"io"
)

// Index is the generated index, built from scanning local source code and version control.
type Index struct {
	Commit plumbing.Hash
	Files  []*File
	Facts  []*Fact
}

// BuildIndex scans the file tree associated with the commit, extracting data from source files
// and building index.
func BuildIndex(commit *object.Commit) (_ *Index, err error) {
	var index = &Index{}

	var tree, _ = commit.Tree()
	var files = tree.Files()

	var scanner = scanner.DockerScanner()
	for {
		var file *object.File
		if file, err = files.Next(); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		index.Files = append(index.Files, &File{Name: file.Name, BlobHash: file.Hash})

		if scanner.Supports(file) {
			var facts []*Fact
			if facts, err = scanner.Scan(file); err != nil {
				return nil, err
			}

			index.Facts = append(index.Facts, facts...)
		}
	}

	return index, nil
}
