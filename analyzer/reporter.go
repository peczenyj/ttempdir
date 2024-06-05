package analyzer

import (
	"go/token"

	"golang.org/x/tools/go/analysis"
)

// Reporter will conclude the ttempdir report.
type Reporter interface {
	Report(fullQualifiedFunctionName string)
}

// ReporterBuilder will prepare a Reporter.
type ReporterBuilder interface {
	Build(position token.Pos) Reporter
	Report(position token.Pos, fullQualifiedFunctionName string)
}

type passReporter struct {
	position token.Pos
	builder  ReporterBuilder
}

func (r *passReporter) Report(fullQualifiedFunctionName string) {
	r.builder.Report(r.position, fullQualifiedFunctionName)
}

type passReporterBuilder struct {
	pass                  *analysis.Pass
	variableOrPackageName string
	targetFunctionName    string
}

func newReporterBuilder(pass *analysis.Pass, variableOrPackageName, targetFunctionName string) ReporterBuilder {
	if variableOrPackageName == "" {
		variableOrPackageName = "testing"
	}

	return &passReporterBuilder{
		pass:                  pass,
		variableOrPackageName: variableOrPackageName,
		targetFunctionName:    targetFunctionName,
	}
}

func (rb *passReporterBuilder) Build(position token.Pos) Reporter {
	return &passReporter{
		position: position,
		builder:  rb,
	}
}

func (rb *passReporterBuilder) Report(position token.Pos, fullQualifiedFunctionName string) {
	rb.pass.Reportf(position,
		"%s() should be replaced by `%s.TempDir()` in %s",
		fullQualifiedFunctionName,
		rb.variableOrPackageName,
		rb.targetFunctionName,
	)
}
