package main

import (
	"github.com/mergestat/kyc"
	"go.riyazali.net/sqlite"
)

// side effect imports for all built-in scanners
import (
	_ "github.com/mergestat/kyc/pkg/scanner/lang/golang"
	_ "github.com/mergestat/kyc/pkg/scanner/meta/files"
	_ "github.com/mergestat/kyc/pkg/scanner/tools/docker"
)

func init() { sqlite.Register(kyc.ExtensionFunc()) }
func main() { /* nothing here fellas */ }
