package kyc

import (
	"encoding/json"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/mergestat/kyc/scanner"
	"go.riyazali.net/sqlite"
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
	var output = &sqlite.IndexInfoOutput{
		ConstraintUsage: make([]*sqlite.ConstraintUsage, len(input.Constraints)),
	}

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

func (cur *FactCursor) Filter(_ int, _ string, _ ...sqlite.Value) (err error) {
	var repo *git.Repository
	if repo, err = git.PlainOpen("."); err != nil {
		return err
	}

	var ref *plumbing.Reference
	if ref, err = repo.Head(); err != nil {
		return err
	}

	var commit *object.Commit
	if commit, err = repo.CommitObject(ref.Hash()); err != nil {
		return err
	}

	var tree, _ = commit.Tree()
	var scanners = scanner.All()

	// TODO(@riyaz): explore async options for file scanning
	// iterate over all files in the tree, and run all scanners against each file
	err = tree.Files().ForEach(func(file *object.File) error {
		for _, scn := range scanners {
			var scannerName = scn.Name()
			if scn.Supports(file) {
				var facts []scanner.Fact
				if facts, err = scn.Scan(file); err != nil {
					return err
				}

				for i := range facts {
					key, val := facts[i].Key, facts[i].Value
					cur.facts = append(cur.facts, &Fact{Commit: commit, File: file, Scanner: scannerName, Key: key, Value: val})
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
