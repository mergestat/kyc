# Know Your Code 🎉

> ⚠️ still experimental! proceed with care!

**`kyc`** is a tool for deriving *facts* about source code in a git repo.
Facts may include:

- The presence (or absence!) of certain files (such as configuration files like `.eslintrc` or `tailwind.config.js` etc.)
- 3rd party dependencies declared in a manifest (such as `go.mod`, `package.json` or `Gemfile`, etc.)
- Structured contents of source code (such as the `FROM` or `RUN` statements in a `Dockerfile`, or the `import`s in a `.go` file)
- ...and more! Please submit an issue if there's a type of fact you'd like to access from your code 🙂

## Goals

The purpose of this project is to enable easy queries on "facts" about git repositories.
There's *a lot* of information tucked away in git repos: in configuration files, source code, commit history, etc, and `kyc` aims to make it easier to extract this information for use cases that involve "knowing your code" better.

It's important to note that the primary purpose of extracting facts in the context of this project is **not** to support IDEs or code navigation - but instead to support users looking to understand the tools, frameworks, libraries, configuration values, versions, structure (and more!) of a git repository for operational purposes.

For running these analyses *across many git repos* we are working on support for `kyc` in [`mergestat`](https://github.com/mergestat/mergestat).

## What does it look like?

Currently, `kyc` is implemented as a SQLite virtual table.
Run the following to build the SQLite extension:

```
go build -o libkyc.so -buildmode=c-shared cmd/shared/shared.go
```

And in a `sqlite3` shell, run

```
sqlite> .load  libkyc.so
```

If you're in the context of a git repository, you can run a query that looks like:

```
sqlite> SELECT * FROM facts WHERE commit_hash = 'bf6fb7e1b42c5f5021fea942c20f3d8c1ff4c2cb';
```

(Where the commit hash is the commit you'd like to derive facts from - `HEAD` will soon be the default).
You'll see a "dump" of all the facts `kyc` has derived from your source code.

To narrow results and present them better, try for example:

```
sqlite> SELECT value->>'path' AS pkg, value->>'version' AS version FROM facts WHERE commit_hash = 'bf6fb7e1b42c5f5021fea942c20f3d8c1ff4c2cb' AND key = '@golang/mod/require';
pkg                                version                           
---------------------------------  ----------------------------------
github.com/bmatcuk/doublestar/v4   v4.6.0                            
github.com/go-git/go-git/v5        v5.6.1                            
github.com/pkg/errors              v0.9.1                            
github.com/smacker/go-tree-sitter  v0.0.0-20230501083651-a7d92773b3aa
go.riyazali.net/sqlite             v0.0.0-20230320080028-80a51d3944c0
golang.org/x/sync                  v0.1.0                            
github.com/Microsoft/go-winio      v0.6.1                            
github.com/ProtonMail/go-crypto    v0.0.0-20230426101702-58e86b294756
github.com/acomagu/bufpipe         v1.0.4                            
github.com/cloudflare/circl        v1.3.2                            
...
```

To list all the dependencies declared in a `go.mod` file and their version.
