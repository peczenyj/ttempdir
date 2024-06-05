# ttempdir

ttempdir detects temporary directories not using t.TempDir

[![tag](https://img.shields.io/github/tag/peczenyj/ttempdir.svg)](https://github.com/peczenyj/ttempdir/releases)
![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.21-%23007d9c)
[![GoDoc](https://pkg.go.dev/badge/github.com/peczenyj/ttempdir)](http://pkg.go.dev/github.com/peczenyj/ttempdir)
[![Go](https://github.com/peczenyj/ttempdir/actions/workflows/go.yml/badge.svg)](https://github.com/peczenyj/ttempdir/actions/workflows/go.yml)
[![Lint](https://github.com/peczenyj/ttempdir/actions/workflows/lint.yml/badge.svg)](https://github.com/peczenyj/ttempdir/actions/workflows/lint.yml)
[![codecov](https://codecov.io/gh/peczenyj/ttempdir/graph/badge.svg?token=9y6f3vGgpr)](https://codecov.io/gh/peczenyj/ttempdir)
[![Report card](https://goreportcard.com/badge/github.com/peczenyj/ttempdir)](https://goreportcard.com/report/github.com/peczenyj/ttempdir)
[![CodeQL](https://github.com/peczenyj/ttempdir/actions/workflows/github-code-scanning/codeql/badge.svg)](https://github.com/peczenyj/ttempdir/actions/workflows/github-code-scanning/codeql)
[![Dependency Review](https://github.com/peczenyj/ttempdir/actions/workflows/dependency-review.yml/badge.svg)](https://github.com/peczenyj/ttempdir/actions/workflows/dependency-review.yml)
[![License](https://img.shields.io/github/license/peczenyj/ttempdir)](./LICENSE)

This code is based on [tenv](https://github.com/sivchari/tenv) analyzer.

## Instruction

```sh
go install github.com/peczenyj/ttempdir/cmd/ttempdir@latest
```

## Usage

```go
package main

import (
    "fmt"
    "io/ioutil"
    "os"
    "testing"
)

func TestMain(t *testing.T) {
    fmt.Println(os.TempDir())
    dir, err := os.MkdirTemp("", "foo")
    if err != nil {
        t.Fatalf("unable to create temporary directory %v", err)
    }
    defer os.RemoveAll(dir)
}

func TestMain2(t *testing.T) {
    fmt.Println(os.TempDir())
}

func helper() {
    dir, err := ioutil.TempDir("", "foo")
    if err != nil {
        panic(fmt.Errorf("unable to create temporary directory: %w", err))
    }
    defer os.RemoveAll(dir)
}
```

```console
go vet -vettool=$(which ttempdir) ./...

# a
./main_test.go:11:14: os.TempDir() should be replaced by `t.TempDir()` in TestMain
./main_test.go:12:2: os.MkdirTemp() should be replaced by `t.TempDir()` in TestMain
./main_test.go:20:14: os.TempDir() should be replaced by `t.TempDir()` in TestMain2
```

### option

The option `all` will run against whole test files (`_test.go`) regardless of method/function signatures.  

By default, only methods that take `*testing.T`, `*testing.B`, and `testing.TB` as arguments are checked.

```go
package main

import (
    "fmt"
    "io/ioutil"
    "os"
    "testing"
)

func TestMain(t *testing.T) {
    fmt.Println(os.TempDir())
    dir, err := os.MkdirTemp("", "foo")
    if err != nil {
        t.Fatalf("unable to create temporary directory %v", err)
    }
    defer os.RemoveAll(dir)
}

func TestMain2(t *testing.T) {
    fmt.Println(os.TempDir())
}

func helper() {
    dir, err := ioutil.TempDir("", "foo")
    if err != nil {
        panic(fmt.Errorf("unable to create temporary directory: %w", err))
    }
    defer os.RemoveAll(dir)
}
```

```console
go vet -vettool=(which ttempdir) -ttempdir.all ./...

# a
./main_test.go:11:14: os.TempDir() should be replaced by `t.TempDir()` in TestMain
./main_test.go:12:2: os.MkdirTemp() should be replaced by `t.TempDir()` in TestMain
./main_test.go:20:14: os.TempDir() should be replaced by `t.TempDir()` in TestMain2
./main_test.go:24:2: ioutil.TempDir() should be replaced by `testing.TempDir()` in helper
```

## CI

### CircleCI

```yaml
- run:
    name: install ttempdir
    command: go install github.com/peczenyj/ttempdir

- run:
    name: run ttempdir
    command: go vet -vettool=`which ttempdir` ./...
```

### GitHub Actions

```yaml
- name: install ttempdir
  run: go install github.com/peczenyj/ttempdir

- name: run ttempdir
  run: go vet -vettool=`which ttempdir` ./...
```
