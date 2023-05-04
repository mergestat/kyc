package docker

import (
	"bytes"
	"context"
	"errors"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/mergestat/kyc/pkg/scanner"
	utils "github.com/mergestat/kyc/pkg/tree-sitter-utils"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/dockerfile"
	"golang.org/x/sync/errgroup"
	"io"
	"strings"
)

type extractFn func(context.Context, *sitter.Tree, []byte, chan<- scanner.Fact) error

// DockerfileScanner implements scanner.Scanner to extract facts from Dockerfiles.
type DockerfileScanner struct{}

func (d *DockerfileScanner) Supports(file *object.File) bool {
	return file.Mode.IsFile() && strings.HasSuffix(file.Name, "Dockerfile")
}

func (d *DockerfileScanner) Scan(ctx context.Context, file *object.File, _ ...string) (_ []scanner.Fact, err error) {
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

	var parser = sitter.NewParser()
	parser.SetLanguage(dockerfile.GetLanguage())

	var tree *sitter.Tree
	if tree, err = parser.ParseCtx(context.Background(), nil, content.Bytes()); err != nil {
		return nil, err
	}
	defer tree.Close()

	var result = make(chan scanner.Fact)
	g, ctx := errgroup.WithContext(ctx)

	var extractors = []extractFn{directiveFrom}
	for _, ext := range extractors {
		g.Go(func() error { return ext(ctx, tree.Copy(), content.Bytes(), result) })
	}

	go func() { err = g.Wait(); close(result) }() // close the channel after all goroutines have returned
	for fact := range result {
		facts = append(facts, fact)
	}

	return facts, err
}

// extractor function to parse FROM directives
func directiveFrom(ctx context.Context, tree *sitter.Tree, content []byte, c chan<- scanner.Fact) error {
	// predicate functions for use with utils.Find()
	isFromInstruction := func(node *sitter.Node) bool { return node.IsNamed() && node.Type() == "from_instruction" }
	isImageSpec := func(node *sitter.Node) bool { return node.IsNamed() && node.Type() == "image_spec" }

	var fromNodes = utils.Find(tree.RootNode(), isFromInstruction) // find all FROM nodes in the tree
	for _, from := range fromNodes {
		var imageSpec = utils.Find(from, isImageSpec)
		if len(imageSpec) != 1 {
			return errors.New("malformed dockerfile")
		}

		var spec = imageSpec[0]
		name, digest, tag := spec.ChildByFieldName("name"), spec.ChildByFieldName("digest"), spec.ChildByFieldName("tag")
		if name == nil || (digest == nil && tag == nil) {
			return errors.New("malformed dockerfile")
		}

		var val = map[string]any{"name": name.Content(content)}
		if digest != nil {
			val["digest"] = digest.Content(content)[1:]
		} else {
			val["tag"] = tag.Content(content)[1:]
		}

		if as := from.ChildByFieldName("as"); as != nil {
			val["alias"] = as.Content(content)
		}

		// emit fact! or not if the ctx is cancelled :)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case c <- scanner.Fact{Key: "@docker/dockerfile/base-image", Value: val}:
		}
	}

	return nil
}

// register the DockerfileScanner with scanner registry
func init() { scanner.Register("docker/dockerfile", &DockerfileScanner{}) }
