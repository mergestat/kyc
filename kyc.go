// Package kyc provides a data extraction library to index software projects.
package kyc

import (
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// Repository represents a `git` source code repository where code for either the whole application or part of it is checked-in.
// kyc understands and works with `git` repository and rely on `git` to provide ownership and changes related metadata and tracking.
type Repository struct {
	r *git.Repository // reference to the underlying git.Repository
}

// Open opens and scan the git repository at the provided path.
func Open(path string) (_ *Repository, err error) {
	var repo *git.Repository
	if repo, err = git.PlainOpen(path); err != nil {
		return nil, err
	}

	return &Repository{r: repo}, nil
}

// ScanHead scans the repository at HEAD, and builds and return an Index.
func (repo *Repository) ScanHead() (_ *Index, err error) {
	var ref *plumbing.Reference
	if ref, err = repo.r.Head(); err != nil {
		return nil, err
	}

	var head *object.Commit
	if head, err = repo.r.CommitObject(ref.Hash()); err != nil {
		return nil, err
	}

	return BuildIndex(head)
}

type File struct {
	Name     string
	BlobHash plumbing.Hash
}

type Fact struct {
	File *File

	Scope string
	Key   string
	Value any
}
