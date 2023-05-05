package kyc

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/bmatcuk/doublestar/v4"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/mergestat/kyc/pkg/scanner"
	"go.riyazali.net/sqlite"
	"os"
)

type Fact struct {
	Commit *object.Commit
	File   *object.File

	Scanner string
	Key     string
	Value   any
}

const (
	ColumnCommit    = iota // hash of the commit in the repository
	ColumnFileName         // name of the file from which the fact was extracted
	ColumnFileBlob         // git blob hash of the file
	ColumnScanner          // name of the scanner used
	ColumnFactKey          // identifier for fact type
	ColumnFactValue        // extracted fact value
)

const (
	_       = iota
	OpEqual // op code for equals-to operation
	OpLike  // op code for LIKE operation
	OpGlob  // op code for GLOB operation
	OpLte   // op code for <= operation
	OpGte   // op code for >= operation
)

// FactModule implements sqlite.Module interface for fact() table-valued function.
type FactModule struct{}

func (mod *FactModule) Connect(_ *sqlite.Conn, _ []string, declare func(string) error) (_ sqlite.VirtualTable, err error) {
	const query = `
		CREATE TABLE facts (
			commit_hash 	TEXT,
			file_name 		TEXT,
			file_blob 		TEXT,
			scanner			TEXT,
			key 			TEXT,
			value
		)`

	if err = declare(query); err != nil {
		return nil, err
	}

	return &FactTable{}, nil
}

// FactTable implements sqlite.VirtualTable interface for fact() table-valued function.
type FactTable struct{}

func (table *FactTable) BestIndex(input *sqlite.IndexInfoInput) (*sqlite.IndexInfoOutput, error) {
	var argv = 1
	var bitmap []byte
	var set = func(op, col int) { bitmap = append(bitmap, byte(op<<4|col)) }

	var output = &sqlite.IndexInfoOutput{
		ConstraintUsage: make([]*sqlite.ConstraintUsage, len(input.Constraints)),
	}

	var commitConstrained = false

	for i, cons := range input.Constraints {
		switch col, op := cons.ColumnIndex, cons.Op; col {
		case ColumnCommit:
			{
				if op != sqlite.INDEX_CONSTRAINT_EQ {
					return nil, sqlite.Error(sqlite.SQLITE_CONSTRAINT, "only equals-to operation is supported on commit hash")
				}

				if cons.Usable {
					output.ConstraintUsage[i] = &sqlite.ConstraintUsage{ArgvIndex: argv, Omit: true}
					commitConstrained, argv = true, argv+1
					set(OpEqual, col)
				}
			}
		case ColumnFileName:
			{
				if op == sqlite.INDEX_CONSTRAINT_GE || op == sqlite.INDEX_CONSTRAINT_LT {
					// sometimes sqlite core try to infer the glob pattern, and uses >= and <
					// operations to hint to the virtual table implementation that it can skip
					// "certain rows" using the range specified by >= and <
					//
					// For example, if we specify "a/**/*.go" as glob pattern, the sqlite core "knows"
					// that we can skip all values that do not start with "a/" and hence it'll use the range
					// operators to signal that.
					//
					// We do not support it at the moment and so we simply ignore these constraints.
					//
					// TODO(@riyaz): look into how we can use these constraints to improve tree traversal performance
					continue
				}

				if op != sqlite.INDEX_CONSTRAINT_EQ && op != sqlite.INDEX_CONSTRAINT_GLOB {
					return nil, sqlite.Error(sqlite.SQLITE_CONSTRAINT, "only equals-to and GLOB operations are supported on file_name")
				}

				if !cons.Usable {
					// TODO(@riyaz): make this error more user-friendly
					return nil, sqlite.Error(sqlite.SQLITE_CONSTRAINT, "file_name constraint must be usable")
				}

				output.ConstraintUsage[i] = &sqlite.ConstraintUsage{ArgvIndex: argv, Omit: true}
				argv += 1

				if op == sqlite.INDEX_CONSTRAINT_EQ {
					set(OpEqual, col)
				} else {
					set(OpGlob, col)
				}
			}
		case ColumnScanner:
			{
				if op != sqlite.INDEX_CONSTRAINT_EQ && op != sqlite.INDEX_CONSTRAINT_LIKE {
					return nil, sqlite.Error(sqlite.SQLITE_CONSTRAINT, "only equals-to and LIKE operations are supported on scanner")
				}

				if !cons.Usable {
					// TODO(@riyaz): make this error more user-friendly
					return nil, sqlite.Error(sqlite.SQLITE_CONSTRAINT, "scanner constraint must be usable")
				}

				// TODO(@riyaz): set omit to true once the constraints are implemented below
				output.ConstraintUsage[i] = &sqlite.ConstraintUsage{ArgvIndex: argv, Omit: false}
				argv += 1

				if op == sqlite.INDEX_CONSTRAINT_EQ {
					set(OpEqual, col)
				} else {
					set(OpLike, col)
				}
			}
		}
	}

	if !commitConstrained {
		return nil, sqlite.Error(sqlite.SQLITE_CONSTRAINT, "commit hash is required")
	}

	// pass the bitmap as string to xFilter routine
	output.IndexString = base64.StdEncoding.EncodeToString(bitmap)

	return output, nil
}

func (table *FactTable) Open() (sqlite.VirtualCursor, error) { return &FactCursor{}, nil }
func (table *FactTable) Disconnect() error                   { return nil }
func (table *FactTable) Destroy() error                      { return nil }

// FactCursor implements sqlite.VirtualCursor interface for fact() table-valued function.
type FactCursor struct {
	pos   int
	facts []*Fact
}

func (cur *FactCursor) Filter(_ int, str string, values ...sqlite.Value) (err error) {
	var ctx = context.Background()

	var cwd string
	if cwd, err = os.Getwd(); err != nil {
		return err
	}

	var repo *git.Repository
	if repo, err = git.PlainOpen(cwd); err != nil {
		return err
	}

	var commit *object.Commit
	var scanners = scanner.All()
	var glob = func(_ *object.File) bool { return true } // default glob func is a no-op

	var bitmap, _ = base64.StdEncoding.DecodeString(str)
	for n, val := range values {
		op, col := (bitmap[n]&0b11110000)>>4, bitmap[n]&0b00001111
		switch {
		case col == ColumnCommit && op == OpEqual:
			{
				if !plumbing.IsHash(val.Text()) {
					return sqlite.Error(sqlite.SQLITE_ERROR, "invalid commit hash")
				}

				if commit, err = repo.CommitObject(plumbing.NewHash(val.Text())); err != nil {
					return sqlite.Error(sqlite.SQLITE_ERROR, err.Error())
				}
			}
		case col == ColumnFileName && (op == OpEqual || op == OpGlob):
			{
				glob = globFunc(val.Text()) // works for both OpEqual and OpGlob
			}
		case col == ColumnScanner:
			{
				// TODO(@riyaz): implement constraint on scanner column
			}
		}

	}

	var tree *object.Tree
	if tree, err = commit.Tree(); err != nil {
		return sqlite.Error(sqlite.SQLITE_ERROR, err.Error())
	}

	// TODO(@riyaz): explore async options for file scanning
	// iterate over all files in the tree, and run all scanners against each file
	err = tree.Files().ForEach(func(file *object.File) error {
		for name, scn := range scanners {
			if glob(file) && scn.Supports(file) {
				var facts []scanner.Fact
				if facts, err = scn.Scan(ctx, file); err != nil {
					return err
				}

				for i := range facts {
					key, val := facts[i].Key, facts[i].Value
					cur.facts = append(cur.facts, &Fact{Commit: commit, File: file, Scanner: name, Key: key, Value: val})
				}
			}
		}

		return nil
	})

	return err
}

func (cur *FactCursor) Column(context *sqlite.VirtualTableContext, pos int) error {
	var fact = cur.facts[cur.pos]

	switch pos {
	case ColumnCommit:
		context.ResultText(fact.Commit.ID().String())
	case ColumnFileName:
		context.ResultText(fact.File.Name)
	case ColumnFileBlob:
		context.ResultText(fact.File.ID().String())
	case ColumnScanner:
		context.ResultText(fact.Scanner)
	case ColumnFactKey:
		context.ResultText(fact.Key)
	case ColumnFactValue:
		var j, _ = json.Marshal(fact.Value)
		context.ResultBlob(j)
		context.ResultSubType(74)
	}
	return nil
}

func (cur *FactCursor) Next() error           { cur.pos += 1; return nil }
func (cur *FactCursor) Rowid() (int64, error) { return int64(cur.pos), nil }
func (cur *FactCursor) Eof() bool             { return cur.pos >= len(cur.facts) }
func (cur *FactCursor) Close() error          { return nil }

// creates a glob pattern matching function
func globFunc(pattern string) func(*object.File) bool {
	return func(file *object.File) bool { var match, _ = doublestar.PathMatch(pattern, file.Name); return match }
}
