package analyzer

import (
	"flag"
	"go/ast"
	"go/token"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const (
	name = "ttempdir"
	doc  = name + " is analyzer that detects using os.MkdirTemp, ioutil.TempDir or os.TempDir instead of t.TempDir since Go1.17"
	url  = "https://github.com/peczenyj/ttempdir"

	defaultAll               = false
	defaultMaxRecursionLevel = 5 // arbitrary value, just to avoid too many recursion calls

	// FlagAllName name of the 'all' flag in cli
	FlagAllName = "all"
	// FlagMaxRecursionLevelName name of the 'max-recursion-level' flag in cli
	FlagMaxRecursionLevelName = "max-recursion-level"
)

type ttempdirAnalyzer struct {
	all               bool
	maxRecursionLevel uint
}

type conf struct {
	prefix string
}

// Option type.
type Option func(*conf)

// WithFlagPrefix functional option.
func WithFlagPrefix(prefix string) Option {
	return func(c *conf) {
		c.prefix = prefix
	}
}

// New analyzer constructor.
// Will bind flagset all and max-recursion-level
func New(opts ...Option) *analysis.Analyzer {
	var c conf

	for _, opt := range opts {
		opt(&c)
	}

	var ta ttempdirAnalyzer

	aa := &analysis.Analyzer{
		Name: name,
		Doc:  doc,
		URL:  url,
		Run:  ta.Run,
		Requires: []*analysis.Analyzer{
			inspect.Analyzer,
		},
	}

	c.bindFlags(&ta, &aa.Flags)

	return aa
}

func (c *conf) bindFlags(ta *ttempdirAnalyzer, flagSet *flag.FlagSet) {
	prefix := c.prefix

	if prefix != "" && !strings.HasSuffix(prefix, ".") {
		prefix += "."
	}

	flagSet.BoolVar(&ta.all,
		prefix+FlagAllName,
		defaultAll,
		"the all option will run against all methods in test file")

	flagSet.UintVar(&ta.maxRecursionLevel,
		prefix+FlagMaxRecursionLevelName,
		defaultMaxRecursionLevel,
		"max recursion level when checking nested arg calls")
}

func (ta *ttempdirAnalyzer) Run(pass *analysis.Pass) (interface{}, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
		(*ast.FuncLit)(nil),
	}

	inspect.Preorder(nodeFilter, func(node ast.Node) {
		ta.checkAstNode(pass, node)
	})

	return nil, nil
}

func (ta *ttempdirAnalyzer) checkAstNode(pass *analysis.Pass, node ast.Node) {
	switch function := node.(type) {
	case *ast.FuncDecl:
		ta.checkFuncDecl(pass, function)
	case *ast.FuncLit:
		ta.checkFuncLit(pass, function, "anonymous function")
	}
}

func (ta *ttempdirAnalyzer) checkFuncDecl(pass *analysis.Pass, function *ast.FuncDecl) {
	ta.checkGenericFunctionCall(pass, function.Type, function.Body, function.Name.Name)
}

func (ta *ttempdirAnalyzer) checkFuncLit(pass *analysis.Pass, function *ast.FuncLit, targetFunctionName string) {
	ta.checkGenericFunctionCall(pass, function.Type, function.Body, targetFunctionName)
}

func (ta *ttempdirAnalyzer) checkGenericFunctionCall(pass *analysis.Pass,
	functionType *ast.FuncType,
	functionBody *ast.BlockStmt,
	targetFunctionName string,
) {
	variableOrPackageName, found := ta.targetRunner(functionType.Params,
		isFilenameFollowingTestingConventions(pass, functionType.Pos()),
	)

	if found {
		reporterBuilder := newReporterBuilder(pass, variableOrPackageName, targetFunctionName)

		ta.checkStmts(reporterBuilder, functionBody.List)
	}
}

func isFilenameFollowingTestingConventions(pass *analysis.Pass, pos token.Pos) bool {
	fileName := pass.Fset.File(pos).Name()

	return strings.HasSuffix(fileName, "_test.go")
}

func (ta *ttempdirAnalyzer) checkStmts(reporterBuilder ReporterBuilder,
	stmts []ast.Stmt,
) {
	for _, stmt := range stmts {
		ta.checkSingleStmt(reporterBuilder, stmt)
	}
}

func (ta *ttempdirAnalyzer) checkSingleStmt(reporterBuilder ReporterBuilder,
	stmt ast.Stmt,
) {
	switch stmt := stmt.(type) {
	case *ast.ExprStmt:
		ta.checkExprStmt(reporterBuilder, stmt)
	case *ast.IfStmt:
		ta.checkIfStmt(reporterBuilder, stmt)
	case *ast.AssignStmt:
		ta.checkAssignStmt(reporterBuilder.Build(stmt.Pos()), stmt)
	case *ast.ForStmt:
		ta.checkForStmt(reporterBuilder, stmt)
	case *ast.DeferStmt:
		ta.checkDeferStmt(reporterBuilder, stmt)
	}
}

func (ta *ttempdirAnalyzer) checkExprStmt(reporterBuilder ReporterBuilder,
	stmt *ast.ExprStmt,
) {
	if callExpr, ok := stmt.X.(*ast.CallExpr); ok {
		ta.checkCallExpr(reporterBuilder, callExpr)
	}
}

func (ta *ttempdirAnalyzer) checkCallExprRecursive(reporterBuilder ReporterBuilder,
	callExpr *ast.CallExpr,
	currentRecursionLevel uint,
) {
	if currentRecursionLevel == 0 {
		return
	}

	currentRecursionLevel--

	for _, arg := range callExpr.Args {
		if argCallExpr, ok := arg.(*ast.CallExpr); ok {
			ta.checkCallExprRecursive(reporterBuilder,
				argCallExpr,
				currentRecursionLevel,
			)
		}
	}

	reporter := reporterBuilder.Build(callExpr.Pos())

	ta.checkFunctionExpr(reporter, callExpr.Fun)
}

func (ta *ttempdirAnalyzer) checkIfStmt(reporterBuilder ReporterBuilder,
	stmt *ast.IfStmt,
) {
	if assignStmt, ok := stmt.Init.(*ast.AssignStmt); ok {
		reporter := reporterBuilder.Build(stmt.Pos())

		ta.checkAssignStmt(reporter, assignStmt)
	}
}

func (ta *ttempdirAnalyzer) checkAssignStmt(reporter Reporter,
	stmt *ast.AssignStmt,
) {
	if rhs, ok := stmt.Rhs[0].(*ast.CallExpr); ok {
		ta.checkFunctionExpr(reporter, rhs.Fun)
	}
}

func (ta *ttempdirAnalyzer) checkDeferStmt(reporterBuilder ReporterBuilder,
	stmt *ast.DeferStmt,
) {
	ta.checkCallExpr(reporterBuilder, stmt.Call)
}

func (ta *ttempdirAnalyzer) checkForStmt(reporterBuilder ReporterBuilder,
	stmt *ast.ForStmt,
) {
	ta.checkStmts(reporterBuilder, stmt.Body.List)
}

func (ta *ttempdirAnalyzer) checkCallExpr(reporterBuilder ReporterBuilder,
	callExpr *ast.CallExpr,
) {
	ta.checkCallExprRecursive(reporterBuilder,
		callExpr,
		ta.maxRecursionLevel,
	)
}

func (ta *ttempdirAnalyzer) checkFunctionExpr(reporter Reporter,
	functionExpr ast.Expr,
) {
	if selectorExpr, ok := functionExpr.(*ast.SelectorExpr); ok {
		ta.checkSelectorExpr(reporter, selectorExpr)
	}
}

func (ta *ttempdirAnalyzer) checkSelectorExpr(reporter Reporter,
	selectorExpr *ast.SelectorExpr,
) {
	if expression, ok := selectorExpr.X.(*ast.Ident); ok {
		ta.checkIdentifiers(reporter, expression, selectorExpr.Sel)
	}
}

func (ta *ttempdirAnalyzer) checkIdentifiers(reporter Reporter,
	expression *ast.Ident,
	fieldSelector *ast.Ident,
) {
	fullQualifiedFunctionName := expression.Name + "." + fieldSelector.Name

	switch fullQualifiedFunctionName {
	case "ioutil.TempDir", "os.MkdirTemp", "os.TempDir":
		reporter.Report(fullQualifiedFunctionName)
	}
}

func (ta *ttempdirAnalyzer) targetRunner(
	functionTypeParams *ast.FieldList,
	isTestFile bool,
) (variableOrPackageName string, found bool) {
	for _, field := range functionTypeParams.List {
		switch typ := field.Type.(type) {
		case *ast.StarExpr:
			if checkStarExprTarget(typ) {
				return getFirstFieldName(field)
			}
		case *ast.SelectorExpr:
			if checkSelectorExprTarget(typ) {
				return getFirstFieldName(field)
			}
		}
	}

	if ta.all && isTestFile {
		return "", true
	}

	return "", false
}

func checkStarExprTarget(typ *ast.StarExpr) bool {
	selector, ok := typ.X.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	x, ok := selector.X.(*ast.Ident)
	if !ok {
		return false
	}
	targetName := x.Name + "." + selector.Sel.Name
	switch targetName {
	case "testing.T", "testing.B", "testing.F":
		return true
	default:
		return false
	}
}

func checkSelectorExprTarget(typ *ast.SelectorExpr) bool {
	x, ok := typ.X.(*ast.Ident)
	if !ok {
		return false
	}
	targetName := x.Name + "." + typ.Sel.Name
	return targetName == "testing.TB"
}

func getFirstFieldName(field *ast.Field) (string, bool) {
	if len(field.Names) == 0 {
		return "", false
	}
	return field.Names[0].Name, true
}
