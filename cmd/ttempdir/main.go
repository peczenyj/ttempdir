package main

import (
	"golang.org/x/tools/go/analysis/unitchecker"

	"github.com/peczenyj/ttempdir"
)

func main() { unitchecker.Main(ttempdir.Analyzer) }
