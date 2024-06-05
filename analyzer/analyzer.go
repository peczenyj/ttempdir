package analyzer

import (
	"flag"
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const (
	doc = "ttempdir is analyzer that detects using os.MkdirTemp, ioutil.TempDir or os.TempDir instead of t.TempDir since Go1.17"

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

// New analyzer constructor.
// Will bind flagset all and max-recursion-level
func New() *analysis.Analyzer {
	ta := &ttempdirAnalyzer{
		all:               defaultAll,
		maxRecursionLevel: defaultMaxRecursionLevel,
	}

	aa := &analysis.Analyzer{
		Name: "ttempdir",
		Doc:  doc,
		Run:  ta.Run,
		Requires: []*analysis.Analyzer{
			inspect.Analyzer,
		},
	}

	ta.Bind(&aa.Flags)

	return aa
}

func (ta *ttempdirAnalyzer) Bind(flagSet *flag.FlagSet) {
	flagSet.BoolVar(&ta.all, FlagAllName, ta.all, "the all option will run against all method in test file")
	flagSet.UintVar(&ta.maxRecursionLevel, FlagMaxRecursionLevelName, ta.maxRecursionLevel, "max recursion level when checking nested arg calls")
}

func (ta *ttempdirAnalyzer) Run(pass *analysis.Pass) (interface{}, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
		(*ast.FuncLit)(nil),
	}

	inspect.Preorder(nodeFilter, func(node ast.Node) {
		switch function := node.(type) {
		case *ast.FuncDecl:
			ta.checkFuncDecl(pass, function)
		case *ast.FuncLit:
			ta.checkFuncLit(pass, function)
		}
	})

	return nil, nil
}

func (ta *ttempdirAnalyzer) checkFuncDecl(pass *analysis.Pass, function *ast.FuncDecl) {
	ta.checkGenericFunctionCall(pass, function, function.Type, function.Body, function.Name.Name)
}

func (ta *ttempdirAnalyzer) checkFuncLit(pass *analysis.Pass, function *ast.FuncLit) {
	ta.checkGenericFunctionCall(pass, function, function.Type, function.Body, "anonymous function")
}

func (ta *ttempdirAnalyzer) checkGenericFunctionCall(pass *analysis.Pass,
	function ast.Node,
	functionType *ast.FuncType,
	functionBody *ast.BlockStmt,
	functionName string,
) {
	fileName := pass.Fset.File(function.Pos()).Name()
	if variableOrPackageName, found := ta.targetRunner(functionType.Params, fileName); found {
		ta.checkStmts(pass, functionBody.List, variableOrPackageName, functionName)
	}
}

func (ta *ttempdirAnalyzer) checkStmts(pass *analysis.Pass,
	stmts []ast.Stmt,
	variableOrPackageName, targetFunctionName string,
) {
	for _, stmt := range stmts {
		ta.checkSingleStmt(pass, stmt, variableOrPackageName, targetFunctionName)
	}
}

func (ta *ttempdirAnalyzer) checkSingleStmt(pass *analysis.Pass,
	stmt ast.Stmt,
	variableOrPackageName, targetFunctionName string,
) {
	switch stmt := stmt.(type) {
	case *ast.ExprStmt:
		ta.checkExprStmt(pass, stmt, variableOrPackageName, targetFunctionName)
	case *ast.IfStmt:
		checkIfStmt(pass, stmt, variableOrPackageName, targetFunctionName)
	case *ast.AssignStmt:
		checkAssignStmt(pass, stmt, variableOrPackageName, targetFunctionName)
	case *ast.ForStmt:
		ta.checkForStmt(pass, stmt, variableOrPackageName, targetFunctionName)
	}
}

func (ta *ttempdirAnalyzer) checkExprStmt(pass *analysis.Pass,
	stmt *ast.ExprStmt,
	variableOrPackageName,
	targetFunctionName string,
) {
	if callExpr, ok := stmt.X.(*ast.CallExpr); ok {
		checkCallExprRecursive(pass,
			callExpr,
			variableOrPackageName,
			targetFunctionName,
			ta.maxRecursionLevel,
		)
	}
}

func checkCallExprRecursive(pass *analysis.Pass,
	callExpr *ast.CallExpr,
	variableOrPackageName, targetFunctionName string,
	currentRecursionLevel uint,
) {
	if currentRecursionLevel == 0 {
		return
	}

	currentRecursionLevel--

	for _, arg := range callExpr.Args {
		if argCallExpr, ok := arg.(*ast.CallExpr); ok {
			checkCallExprRecursive(pass,
				argCallExpr,
				variableOrPackageName,
				targetFunctionName,
				currentRecursionLevel,
			)
		}
	}

	checkFunctionExpr(pass, callExpr, callExpr.Fun, variableOrPackageName, targetFunctionName)
}

func checkIfStmt(pass *analysis.Pass,
	stmt *ast.IfStmt,
	variableOrPackageName, targetFunctionName string,
) {
	if assignStmt, ok := stmt.Init.(*ast.AssignStmt); ok {
		checkAssignStmt(pass, assignStmt, variableOrPackageName, targetFunctionName)
	}
}

func checkAssignStmt(pass *analysis.Pass,
	stmt *ast.AssignStmt,
	variableOrPackageName, targetFunctionName string,
) {
	if rhs, ok := stmt.Rhs[0].(*ast.CallExpr); ok {
		checkFunctionExpr(pass, stmt, rhs.Fun, variableOrPackageName, targetFunctionName)
	}
}

func (ta *ttempdirAnalyzer) checkForStmt(pass *analysis.Pass,
	stmt *ast.ForStmt,
	variableOrPackageName, targetFunctionName string,
) {
	ta.checkStmts(pass, stmt.Body.List, variableOrPackageName, targetFunctionName)
}

func checkFunctionExpr(pass *analysis.Pass,
	stmt ast.Node,
	functionExpr ast.Expr,
	variableOrPackageName, targetFunctionName string,
) {
	if selectorExpr, ok := functionExpr.(*ast.SelectorExpr); ok {
		checkSelectorExpr(pass, stmt, selectorExpr, variableOrPackageName, targetFunctionName)
	}
}

func checkSelectorExpr(pass *analysis.Pass,
	stmt ast.Node,
	selectorExpr *ast.SelectorExpr,
	variableOrPackageName, targetFunctionName string,
) {
	if expression, ok := selectorExpr.X.(*ast.Ident); ok {
		checkIdentifiers(pass, stmt, expression, selectorExpr.Sel, variableOrPackageName, targetFunctionName)
	}
}

func checkIdentifiers(pass *analysis.Pass,
	stmt ast.Node,
	expression *ast.Ident,
	fieldSelector *ast.Ident,
	variableOrPackageName, targetFunctionName string,
) {
	fullQualifiedFunctionName := expression.Name + "." + fieldSelector.Name

	switch fullQualifiedFunctionName {
	case "ioutil.TempDir", "os.MkdirTemp", "os.TempDir":
		if variableOrPackageName == "" {
			variableOrPackageName = "testing"
		}
		pass.Reportf(stmt.Pos(), "%s() should be replaced by `%s.TempDir()` in %s",
			fullQualifiedFunctionName, variableOrPackageName, targetFunctionName)
	}
}

func (ta *ttempdirAnalyzer) targetRunner(
	functionTypeParams *ast.FieldList,
	fileName string,
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

	if ta.all && strings.HasSuffix(fileName, "_test.go") {
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
