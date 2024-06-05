package main

import (
	"golang.org/x/tools/go/analysis/unitchecker"

	"github.com/peczenyj/ttempdir/analyzer"
)

func main() { unitchecker.Main(analyzer.New()) }
