package kyc

import (
	"encoding/base64"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"io"
	"os"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	//"github.com/mergestat/mergestat-lite/pkg/mailmap"
	"github.com/pkg/errors"
	"go.riyazali.net/sqlite"
)

const (
	ColumnCommitHash        = iota // hash of the commit in the repository
	ColumnCommitMessage            // user supplied commit message
	ColumnCommitAuthorName         // name of the commit author
	ColumnCommitAuthorEmail        // email of the commit author
	ColumnCommitAuthorWhen         // datetime when the commit was made
	ColumnCommitParents            // number of parents of the commit
)

// CommitsModule implements sqlite.Module interface for commits() table-valued function.
type CommitsModule struct{}

func (mod *CommitsModule) Connect(_ *sqlite.Conn, _ []string, declare func(string) error) (_ sqlite.VirtualTable, err error) {
	const schema = `
		CREATE TABLE commits (
			hash 			TEXT,
			message 		TEXT,
			author_name 	TEXT,
			author_email 	TEXT,
			author_when	 	DATETIME, 
			parents 		INT,
			
			PRIMARY KEY ( hash )
		) WITHOUT ROWID`

	if err = declare(schema); err != nil {
		return nil, err
	}

	return &CommitsTable{}, nil
}

// CommitsTable implements sqlite.VirtualTable interface for commits() table-valued function.
type CommitsTable struct{}

func (tab *CommitsTable) BestIndex(input *sqlite.IndexInfoInput) (*sqlite.IndexInfoOutput, error) {
	var argv = 0
	var bitmap []byte
	var set = func(op, col int) { bitmap = append(bitmap, byte(op<<4|col)) }

	var out = &sqlite.IndexInfoOutput{
		ConstraintUsage: make([]*sqlite.ConstraintUsage, len(input.Constraints)),
	}

	for i, cons := range input.Constraints {
		switch col, op := cons.ColumnIndex, cons.Op; col {
		// user has specified WHERE hash = 'xxx' .. we just need to pick a single commit here
		case ColumnCommitHash:
			{
				if op == sqlite.INDEX_CONSTRAINT_EQ && cons.Usable {
					set(OpEqual, col)
					out.ConstraintUsage[i] = &sqlite.ConstraintUsage{ArgvIndex: argv}
					out.EstimatedCost, out.EstimatedRows = 1, 1
					out.IdxFlags |= sqlite.INDEX_SCAN_UNIQUE // we only visit at most one row or commit
				}
			}

		// user has specified <= or  => constraint on when column
		case ColumnCommitAuthorWhen:
			{
				out.ConstraintUsage[i] = &sqlite.ConstraintUsage{ArgvIndex: argv, Omit: true}
				if op == sqlite.INDEX_CONSTRAINT_LE {
					set(OpLte, col)
				} else if op == sqlite.INDEX_CONSTRAINT_GE {
					set(OpGte, col)
				}
			}
		}
	}

	// since we already return the commits ordered by descending order of commit time
	// if the user specifies an ORDER BY when DESC we can signal to sqlite3
	// that the output would already be ordered, so it doesn't have to program a separate sort routine
	if len(input.OrderBy) == 1 && input.OrderBy[0].ColumnIndex == ColumnCommitAuthorWhen && input.OrderBy[0].Desc {
		out.OrderByConsumed = true
	}

	out.IndexString = base64.StdEncoding.EncodeToString(bitmap)

	return out, nil
}

func (tab *CommitsTable) Open() (sqlite.VirtualCursor, error) { return &CommitsCursor{}, nil }
func (tab *CommitsTable) Disconnect() error                   { return nil }
func (tab *CommitsTable) Destroy() error                      { return nil }

// CommitsCursor implements sqlite.VirtualCursor interface for commits() table-valued function.
type CommitsCursor struct {
	repo *git.Repository
	rev  *plumbing.Revision

	commit  *object.Commit // the current commit
	commits object.CommitIter
}

func (cur *CommitsCursor) Filter(_ int, s string, values ...sqlite.Value) (err error) {
	var cwd string
	if cwd, err = os.Getwd(); err != nil {
		return err
	}

	var repo *git.Repository
	if repo, err = git.PlainOpen(cwd); err != nil {
		return err
	}

	var opts = &git.LogOptions{Order: git.LogOrderDefault}

	var bitmap, _ = base64.StdEncoding.DecodeString(s)
	for n, val := range values {
		op, col := (bitmap[n]&0b11110000)>>4, bitmap[n]&0b00001111
		switch {
		case col == ColumnCommitHash && op == OpEqual:
			{
				// we only need to get a single commit
				var hash = val.Text()
				if !plumbing.IsHash(hash) {
					return sqlite.Error(sqlite.SQLITE_ERROR, "invalid commit hash")
				}

				cur.commits = object.NewCommitIter(repo.Storer, storer.NewEncodedObjectLookupIter(
					repo.Storer, plumbing.CommitObject, []plumbing.Hash{plumbing.NewHash(hash)}))

				return cur.Next()
			}

		case col == ColumnCommitAuthorWhen:
			if op == OpLte {
				var t time.Time
				if t, err = time.Parse(time.RFC3339, val.Text()); err != nil {
					return sqlite.Error(sqlite.SQLITE_ERROR, "invalid time format")
				}
				opts.Until = &t
			} else if op == OpGte {
				var t time.Time
				if t, err = time.Parse(time.RFC3339, val.Text()); err != nil {
					return sqlite.Error(sqlite.SQLITE_ERROR, "invalid time format")
				}
				opts.Since = &t
			}
		}
	}

	// TODO(@riyaz): add support for non-HEAD references
	var head *plumbing.Reference
	if head, err = repo.Head(); err != nil {
		return errors.Wrapf(err, "failed to resolve head")
	}
	opts.From = head.Hash()

	if cur.commits, err = repo.Log(opts); err != nil {
		return errors.Wrap(err, "failed to create iterator")
	}

	return cur.Next()
}

func (cur *CommitsCursor) Column(c *sqlite.VirtualTableContext, col int) error {
	commit := cur.commit

	switch col {
	case ColumnCommitHash:
		c.ResultText(commit.Hash.String())
	case ColumnCommitMessage:
		c.ResultText(commit.Message)
	case ColumnCommitAuthorName:
		c.ResultText(commit.Author.Name)
	case ColumnCommitAuthorEmail:
		c.ResultText(commit.Author.Email)
	case ColumnCommitAuthorWhen:
		c.ResultText(commit.Author.When.Format(time.RFC3339))
	case ColumnCommitParents:
		c.ResultInt(commit.NumParents())
	}

	return nil
}

func (cur *CommitsCursor) Next() (err error) {
	if cur.commit, err = cur.commits.Next(); err != nil {
		// check for ErrObjectNotFound to ensure we don't crash
		// if the user provided hash did not point to a commit
		if err != io.EOF && err != plumbing.ErrObjectNotFound {
			return err
		}
	}
	return nil
}

func (cur *CommitsCursor) Eof() bool             { return cur.commit == nil }
func (cur *CommitsCursor) Rowid() (int64, error) { return int64(0), nil }
func (cur *CommitsCursor) Close() error {
	if cur.commits != nil {
		cur.commits.Close()
	}
	return nil
}
