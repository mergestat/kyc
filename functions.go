package kyc

import (
	"bytes"
	"fmt"
	"github.com/ghodss/yaml"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/pkg/errors"
	"go.riyazali.net/sqlite"
	"io"
)

// YamlToJson implements yaml_to_json sql function.
// The function signature of the equivalent sql function is:
//
//	yaml_to_json(string) string
type YamlToJson struct{}

func (y *YamlToJson) Args() int           { return 1 }
func (y *YamlToJson) Deterministic() bool { return true }

func (y *YamlToJson) Apply(context *sqlite.Context, value ...sqlite.Value) {
	if json, err := yaml.YAMLToJSON(value[0].Blob()); err != nil {
		context.ResultError(err)
	} else {
		context.ResultText(string(json))
	}
}

// ReadBlob implements read_blob() sql function that reads
// the content of the blob identified by the provided hash from the given repository.
type ReadBlob struct{}

func (r *ReadBlob) Deterministic() bool { return true }
func (r *ReadBlob) Args() int           { return 2 }

func (r *ReadBlob) Apply(context *sqlite.Context, values ...sqlite.Value) {
	var ok bool
	var err error

	var repo *git.Repository
	if repo, ok = values[0].Pointer().(*git.Repository); !ok {
		context.ResultError(fmt.Errorf("first argument must be a pointer to a git repository"))
		return
	}

	if !plumbing.IsHash(values[1].Text()) {
		context.ResultError(fmt.Errorf("second value is not a valid hash"))
		return
	}

	var hash = plumbing.NewHash(values[1].Text())
	var blob *object.Blob
	if blob, err = repo.BlobObject(hash); err != nil {
		if err == plumbing.ErrObjectNotFound {
			context.ResultError(fmt.Errorf("blob with hash %q not found", hash.String()))
		} else if err == plumbing.ErrInvalidType {
			context.ResultError(fmt.Errorf("object with hash %q is not a blob", hash.String()))
		} else {
			context.ResultError(err)
		}
		return
	}

	var reader io.ReadCloser
	if reader, err = blob.Reader(); err != nil {
		context.ResultError(errors.Wrapf(err, "failed to read blob"))
		return
	}
	defer reader.Close()

	var buf bytes.Buffer
	if _, err = io.Copy(&buf, reader); err != nil {
		context.ResultError(errors.Wrapf(err, "failed to read blob"))
		return
	}

	context.ResultBlob(buf.Bytes())
}
