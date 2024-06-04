package ttempdir_test

import (
	"flag"
	"testing"

	"github.com/gostaticanalysis/testutil"
	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/peczenyj/ttempdir"
)

// TestAnalyzer is a test for Analyzer.
func TestAnalyzer(t *testing.T) {
	defaults := func() map[string]string {
		return map[string]string{
			ttempdir.A:   "false",
			ttempdir.MRL: "5",
		}
	}

	t.Cleanup(func() {
		setKV(t, &ttempdir.Analyzer.Flags, defaults())
	})

	testcases := []struct {
		label    string
		flags    map[string]string
		patterns []string
	}{
		{
			label:    "default flags",
			flags:    defaults(),
			patterns: []string{"a", "b", "c"},
		},
		{
			label: "flag all=true",
			flags: map[string]string{
				ttempdir.A:   "true",
				ttempdir.MRL: "5",
			},
			patterns: []string{"d"},
		},
		{
			label: "flag max-recursion-level=10",
			flags: map[string]string{
				ttempdir.A:   "false",
				ttempdir.MRL: "10",
			},
			patterns: []string{"e"},
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.label, func(t *testing.T) {
			testdata := testutil.WithModules(t, analysistest.TestData(), nil)
			analyzer := ttempdir.Analyzer

			setKV(t, &analyzer.Flags, tc.flags)

			analysistest.Run(t, testdata, analyzer, tc.patterns...)
		})
	}
}

func setKV(t *testing.T, flagset *flag.FlagSet, flags map[string]string) {
	t.Helper()

	for k, v := range flags {
		err := flagset.Set(k, v)
		if err != nil {
			t.Fatalf("unable to set k %q v %q: %v", k, v, err)
		}
	}
}
