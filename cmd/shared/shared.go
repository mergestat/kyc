package main

import (
	"github.com/mergestat/kyc"
	"go.riyazali.net/sqlite"
)

func init() { sqlite.Register(kyc.ExtensionFunc()) }
func main() { /* nothing here fellas */ }
