package analyzer

import (
	"go/token"

	"golang.org/x/tools/go/analysis"
)

type passReporter struct {
	builder  *passReporterBuilder
	position token.Pos
}

func (r *passReporter) Report(fullQualifiedFunctionName string) {
	r.builder.Report(r.position, fullQualifiedFunctionName)
}

type passReporterBuilder struct {
	pass                  *analysis.Pass
	variableOrPackageName string
	targetFunctionName    string
}

func newReporterBuilder(pass *analysis.Pass,
	variableOrPackageName, targetFunctionName string,
) *passReporterBuilder {
	if variableOrPackageName == "" {
		variableOrPackageName = "testing"
	}

	return &passReporterBuilder{
		pass:                  pass,
		variableOrPackageName: variableOrPackageName,
		targetFunctionName:    targetFunctionName,
	}
}

func (rb *passReporterBuilder) Build(position token.Pos) *passReporter {
	return &passReporter{
		position: position,
		builder:  rb,
	}
}

func (rb *passReporterBuilder) Report(position token.Pos,
	fullQualifiedFunctionName string,
) {
	rb.pass.Reportf(position,
		"%s() should be replaced by `%s.TempDir()` in %s",
		fullQualifiedFunctionName,
		rb.variableOrPackageName,
		rb.targetFunctionName,
	)
}
