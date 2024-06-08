package main

import (
	"golang.org/x/tools/go/analysis/singlechecker"

	"github.com/peczenyj/ttempdir/analyzer"
)

func main() { singlechecker.Main(analyzer.New(analyzer.WithFlagPrefix("linter"))) }
