package analyzer_test

import (
	"testing"

	"github.com/gostaticanalysis/testutil"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/peczenyj/ttempdir/analyzer"
)

// TestAnalyzer is a test for Analyzer.
func TestAnalyzer(t *testing.T) {
	testcases := []struct {
		label    string
		flags    map[string]string
		patterns []string
	}{
		{
			label:    "default flags",
			patterns: []string{"a", "b", "c"},
		},
		{
			label: "flag all=true",
			flags: map[string]string{
				analyzer.FlagAllName: "true",
			},
			patterns: []string{"d"},
		},
		{
			label: "flag max-recursion-level=10",
			flags: map[string]string{
				analyzer.FlagMaxRecursionLevelName: "10",
			},
			patterns: []string{"e"},
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.label, func(t *testing.T) {
			testdata := testutil.WithModules(t, analysistest.TestData(), nil)

			ttempdirAnalyze := analyzer.New()

			setKV(t, ttempdirAnalyze, tc.flags)

			analysistest.Run(t, testdata, ttempdirAnalyze, tc.patterns...)
		})
	}
}

func setKV(t *testing.T, instance *analysis.Analyzer, flags map[string]string) {
	t.Helper()

	for k, v := range flags {
		err := instance.Flags.Set(k, v)
		if err != nil {
			t.Fatalf("unable to set k %q v %q: %v", k, v, err)
		}
	}
}
