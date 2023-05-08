package golang

import (
	"bytes"
	"context"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/mergestat/kyc/pkg/scanner"
	"golang.org/x/mod/modfile"
	"io"
)

type GoMod struct{}

func (g *GoMod) Supports(file *object.File) bool {
	return file.Mode.IsFile() && file.Name == "go.mod"
}

func (g *GoMod) Scan(ctx context.Context, file *object.File, _ ...string) (_ []scanner.Fact, err error) {
	var facts []scanner.Fact

	// read the file content to parse
	var reader io.ReadCloser
	if reader, err = file.Reader(); err != nil {
		return nil, err
	}
	defer reader.Close()

	var content bytes.Buffer
	if _, err = io.Copy(&content, reader); err != nil {
		return nil, err
	}

	var module *modfile.File
	if module, err = modfile.ParseLax(file.Name, content.Bytes(), nil /* version fixer */); err != nil {
		return nil, err
	}

	// for each require, emit a fact
	for _, req := range module.Require {
		var val = map[string]any{"path": req.Mod.Path, "version": req.Mod.Version}
		facts = append(facts, scanner.Fact{Key: "@golang/mod/require", Value: val})
	}

	return facts, nil
}

// register the GoMod with scanner registry
func init() { scanner.Register("golang/mod", &GoMod{}) }
