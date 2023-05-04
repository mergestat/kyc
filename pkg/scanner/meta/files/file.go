package files

import (
	"context"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/mergestat/kyc/pkg/scanner"
)

// FileMeta implements scanner.Scanner that emit facts about the file being scanned
type FileMeta struct{}

func (f *FileMeta) Supports(_ *object.File) bool { return true }

func (f *FileMeta) Scan(_ context.Context, file *object.File, _ ...string) ([]scanner.Fact, error) {
	var val = map[string]any{"name": file.Name, "mode": file.Mode.String()}
	return []scanner.Fact{{Key: "@files/meta", Value: val}}, nil
}

func init() { scanner.Register("files", &FileMeta{}) }
