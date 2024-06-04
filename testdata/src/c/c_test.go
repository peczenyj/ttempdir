package c

import (
	"os"
	"testing"
)

var (
	tdir = os.TempDir() // never seen
)

func testsetup() {
	os.TempDir()        // if -all = true, want  "os\\.TempDir\\(\\) can be replaced by `testing\\.TempDir\\(\\)` in testsetup"
	dir := os.TempDir() // if -all = true, want  "os\\.TempDir\\(\\) can be replaced by `testing\\.TempDir\\(\\)` in testsetup"
	_ = dir
	_ = os.TempDir() // if -all = true, "func setup is not using testing.TempDir"
}

func TestF(t *testing.T) {
	testsetup()
	os.TempDir()                       // want "os\\.TempDir\\(\\) can be replaced by `t\\.TempDir\\(\\)` in TestF"
	_ = os.TempDir()                   // want "os\\.TempDir\\(\\) can be replaced by `t\\.TempDir\\(\\)` in TestF"
	if dir = os.TempDir(); dir != "" { // want "os\\.TempDir\\(\\) can be replaced by `t\\.TempDir\\(\\)` in TestF"
		_ = dir
	}
}

func BenchmarkF(b *testing.B) {
	TB(b)
	os.TempDir()                       // want "os\\.TempDir\\(\\) can be replaced by `b\\.TempDir\\(\\)` in BenchmarkF"
	_ = os.TempDir()                   // want "os\\.TempDir\\(\\) can be replaced by `b\\.TempDir\\(\\)` in BenchmarkF"
	if dir = os.TempDir(); dir != "" { // want "os\\.TempDir\\(\\) can be replaced by `b\\.TempDir\\(\\)` in BenchmarkF"
		_ = dir
	}
}

func TB(tb testing.TB) {
	os.TempDir()                       // want "os\\.TempDir\\(\\) can be replaced by `tb\\.TempDir\\(\\)` in TB"
	_ = os.TempDir()                   // want "os\\.TempDir\\(\\) can be replaced by `tb\\.TempDir\\(\\)` in TB"
	if dir = os.TempDir(); dir != "" { // want "os\\.TempDir\\(\\) can be replaced by `tb\\.TempDir\\(\\)` in TB"
		_ = dir
	}
}

func FuzzF(f *testing.F) {
	os.TempDir()                       // want "os\\.TempDir\\(\\) can be replaced by `f\\.TempDir\\(\\)` in FuzzF"
	_ = os.TempDir()                   // want "os\\.TempDir\\(\\) can be replaced by `f\\.TempDir\\(\\)` in FuzzF"
	if dir = os.TempDir(); dir != "" { // want "os\\.TempDir\\(\\) can be replaced by `f\\.TempDir\\(\\)` in FuzzF"
		_ = dir
	}
}

func TestFunctionLiteral(t *testing.T) {
	testsetup()
	t.Run("test", func(t *testing.T) {
		os.TempDir()                       // want "os\\.TempDir\\(\\) can be replaced by `t\\.TempDir\\(\\)` in anonymous function"
		_ = os.TempDir()                   // want "os\\.TempDir\\(\\) can be replaced by `t\\.TempDir\\(\\)` in anonymous function"
		if dir = os.TempDir(); dir != "" { // want "os\\.TempDir\\(\\) can be replaced by `t\\.TempDir\\(\\)` in anonymous function"
			_ = dir
		}
	})
}

func TestEmpty(t *testing.T) {
	t.Run("test", func(*testing.T) {})
}

func TestEmptyTB(t *testing.T) {
	func(testing.TB) {}(t)
}
