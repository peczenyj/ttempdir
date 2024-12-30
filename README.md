# ttempdir

ttempdir detects temporary directories not using t.TempDir

[![tag](https://img.shields.io/github/tag/peczenyj/ttempdir.svg)](https://github.com/peczenyj/ttempdir/releases)
![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.22.9-%23007d9c)
[![GoDoc](https://pkg.go.dev/badge/github.com/peczenyj/ttempdir)](http://pkg.go.dev/github.com/peczenyj/ttempdir)
[![Go](https://github.com/peczenyj/ttempdir/actions/workflows/go.yml/badge.svg)](https://github.com/peczenyj/ttempdir/actions/workflows/go.yml)
[![Lint](https://github.com/peczenyj/ttempdir/actions/workflows/lint.yml/badge.svg)](https://github.com/peczenyj/ttempdir/actions/workflows/lint.yml)
[![codecov](https://codecov.io/gh/peczenyj/ttempdir/graph/badge.svg?token=9y6f3vGgpr)](https://codecov.io/gh/peczenyj/ttempdir)
[![Report card](https://goreportcard.com/badge/github.com/peczenyj/ttempdir)](https://goreportcard.com/report/github.com/peczenyj/ttempdir)
[![CodeQL](https://github.com/peczenyj/ttempdir/actions/workflows/github-code-scanning/codeql/badge.svg)](https://github.com/peczenyj/ttempdir/actions/workflows/github-code-scanning/codeql)
[![Dependency Review](https://github.com/peczenyj/ttempdir/actions/workflows/dependency-review.yml/badge.svg)](https://github.com/peczenyj/ttempdir/actions/workflows/dependency-review.yml)
[![License](https://img.shields.io/github/license/peczenyj/ttempdir)](./LICENSE)
[![Latest release](https://img.shields.io/github/release/peczenyj/ttempdir.svg)](https://github.com/peczenyj/ttempdir/releases/latest)
[![GitHub Release Date](https://img.shields.io/github/release-date/peczenyj/ttempdir.svg)](https://github.com/peczenyj/ttempdir/releases/latest)
[![Last commit](https://img.shields.io/github/last-commit/peczenyj/ttempdir.svg)](https://github.com/peczenyj/ttempdir/commit/HEAD)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](https://github.com/peczenyj/ttempdir/blob/main/CONTRIBUTING.md#pull-request-process)

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
$ ttempdir ./...

./main_test.go:11:14: os.TempDir() should be replaced by `t.TempDir()` in TestMain
./main_test.go:12:2: os.MkdirTemp() should be replaced by `t.TempDir()` in TestMain
./main_test.go:20:14: os.TempDir() should be replaced by `t.TempDir()` in TestMain2
```

### options

This linter defines two option flags: `-linter.all` and `-linter.max-recursion-level`

```console
$ ttempdir -h
...
  -linter.all
        the all option will run against all methods in test file
  -linter.max-recursion-level uint
        max recursion level when checking nested arg calls (default 5)
...
```

#### all

The option `all` will run against whole test files (`_test.go`) regardless of method/function signatures.  

It is triggered by the flag `-linter.all`.

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
$ ttempdir -linter.all ./...

# a
./main_test.go:11:14: os.TempDir() should be replaced by `t.TempDir()` in TestMain
./main_test.go:12:2: os.MkdirTemp() should be replaced by `t.TempDir()` in TestMain
./main_test.go:20:14: os.TempDir() should be replaced by `t.TempDir()` in TestMain2
./main_test.go:24:2: ioutil.TempDir() should be replaced by `testing.TempDir()` in helper
```

#### max-recursion-level

This linter searches on argument lists in a recursive way. By default we limit to 5 the recursion level.

For instance, the example below will not emit any analysis report because `os.TempDir()` is called on a 6th level of recursion. If needed this can be updated via flag `-linter.max-recursion-level`.

```go
    t.Log( // recursion level 1
        fmt.Sprintf("%s/foo-%d", // recursion level 2
            filepath.Join( // recursion level 3
                filepath.Clean( // recursion level 4
                    fmt.Sprintf("%s", // recursion level 5
                        os.TempDir(), // max recursion level reached.
                    ),
                ),
                "test",
            ),
            1024,
        ),
    )
```

## CI

### CircleCI

```yaml
- run:
    name: install ttempdir
    command: go install github.com/peczenyj/ttempdir/cmd/ttempdir@latest

- run:
    name: run ttempdir
    command: ttempdir ./...
```

### GitHub Actions

```yaml
- name: install ttempdir
  run: go install github.com/peczenyj/ttempdir/cmd/ttempdir@latest

- name: run ttempdir
  run: ttempdir ./...
```
