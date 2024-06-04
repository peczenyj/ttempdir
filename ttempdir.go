package ttempdir

import (
	"go/ast"
	"go/token"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const (
	doc = "ttempdir is analyzer that detects using os.MkdirTemp, ioutil.TempDir or os.TempDir instead of t.TempDir since Go1.17"

	defaultMaxRecursionLevel = 5 // arbitrary value, just to avoid too many recursion calls
)

// Analyzer is ttempdir analyzer
var Analyzer = &analysis.Analyzer{
	Name: "ttempdir",
	Doc:  doc,
	Run:  run,
	Requires: []*analysis.Analyzer{
		inspect.Analyzer,
	},
}

var (
	A       = "all"
	aflag   bool
	MRL     = "max-recursion-level"
	mrlFlag uint
)

func init() {
	Analyzer.Flags.BoolVar(&aflag, A, false, "the all option will run against all method in test file")
	Analyzer.Flags.UintVar(&mrlFlag, MRL, defaultMaxRecursionLevel, "max recursion level when checking nested arg calls")
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
		(*ast.FuncLit)(nil),
	}

	inspect.Preorder(nodeFilter, func(n ast.Node) {
		switch n := n.(type) {
		case *ast.FuncDecl:
			checkFuncDecl(pass, n, pass.Fset.File(n.Pos()).Name())
		case *ast.FuncLit:
			checkFuncLit(pass, n, pass.Fset.File(n.Pos()).Name())
		}
	})

	return nil, nil
}

func checkFuncDecl(pass *analysis.Pass, f *ast.FuncDecl, fileName string) {
	argName, ok := targetRunner(f.Type.Params.List, fileName)
	if !ok {
		return
	}
	checkStmts(pass, f.Body.List, f.Name.Name, argName)
}

func checkFuncLit(pass *analysis.Pass, f *ast.FuncLit, fileName string) {
	argName, ok := targetRunner(f.Type.Params.List, fileName)
	if !ok {
		return
	}
	checkStmts(pass, f.Body.List, "anonymous function", argName)
}

func checkStmts(pass *analysis.Pass, stmts []ast.Stmt, funcName, argName string) {
	for _, stmt := range stmts {
		switch stmt := stmt.(type) {
		case *ast.ExprStmt:
			if !checkExprStmt(pass, stmt, funcName, argName) {
				continue
			}
		case *ast.IfStmt:
			if !checkIfStmt(pass, stmt, funcName, argName) {
				continue
			}
		case *ast.AssignStmt:
			if !checkAssignStmt(pass, stmt, funcName, argName) {
				continue
			}
		}
	}
}

func checkExprStmt(pass *analysis.Pass, stmt *ast.ExprStmt, funcName, argName string) bool {
	callExpr, ok := stmt.X.(*ast.CallExpr)
	if !ok {
		return false
	}

	checkCallExprRecursive(pass, callExpr, funcName, argName, mrlFlag)

	return true
}

func checkCallExprRecursive(pass *analysis.Pass,
	callExpr *ast.CallExpr,
	funcName, argName string,
	currentRecursionLevel uint,
) {
	if currentRecursionLevel == 0 {
		return
	}

	for _, arg := range callExpr.Args {
		if argCallExpr, ok := arg.(*ast.CallExpr); ok {
			checkCallExprRecursive(pass, argCallExpr, funcName, argName, currentRecursionLevel-1)
		}
	}

	fun, ok := callExpr.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}
	x, ok := fun.X.(*ast.Ident)
	if !ok {
		return
	}

	checkTargetNames(pass, callExpr, funcName, argName, fun, x)
}

func checkIfStmt(pass *analysis.Pass, stmt *ast.IfStmt, funcName, argName string) bool {
	assignStmt, ok := stmt.Init.(*ast.AssignStmt)
	if !ok {
		return false
	}
	rhs, ok := assignStmt.Rhs[0].(*ast.CallExpr)
	if !ok {
		return false
	}
	fun, ok := rhs.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	x, ok := fun.X.(*ast.Ident)
	if !ok {
		return false
	}

	checkTargetNames(pass, stmt, funcName, argName, fun, x)

	return true
}

func checkAssignStmt(pass *analysis.Pass, stmt *ast.AssignStmt, funcName, argName string) bool {
	rhs, ok := stmt.Rhs[0].(*ast.CallExpr)
	if !ok {
		return false
	}
	fun, ok := rhs.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	x, ok := fun.X.(*ast.Ident)
	if !ok {
		return false
	}

	checkTargetNames(pass, stmt, funcName, argName, fun, x)

	return true
}

func checkTargetNames(pass *analysis.Pass,
	stmt interface{ Pos() token.Pos },
	funcName, argName string,
	fun *ast.SelectorExpr,
	x *ast.Ident,
) {
	targetName := x.Name + "." + fun.Sel.Name

	switch targetName {
	case "ioutil.TempDir", "os.MkdirTemp", "os.TempDir":
		if argName == "" {
			argName = "testing"
		}
		pass.Reportf(stmt.Pos(), "%s() can be replaced by `%s.TempDir()` in %s", targetName, argName, funcName)

		return
	}
}

func targetRunner(params []*ast.Field, fileName string) (string, bool) {
	for _, p := range params {
		switch typ := p.Type.(type) {
		case *ast.StarExpr:
			if checkStarExprTarget(typ) {
				if len(p.Names) == 0 {
					return "", false
				}
				argName := p.Names[0].Name
				return argName, true
			}
		case *ast.SelectorExpr:
			if checkSelectorExprTarget(typ) {
				if len(p.Names) == 0 {
					return "", false
				}
				argName := p.Names[0].Name
				return argName, true
			}
		}
	}
	if aflag && strings.HasSuffix(fileName, "_test.go") {
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
