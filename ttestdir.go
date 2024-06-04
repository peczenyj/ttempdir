package ttempdir

import (
	"go/ast"
	"go/token"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const doc = "ttempdir is analyzer that detects using os.MkdirTemp or ioutil.TempDir instead of t.TempDir since Go1.15"

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
	A     = "all"
	aflag bool
)

func init() {
	Analyzer.Flags.BoolVar(&aflag, A, false, "the all option will run against all method in test file")
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
		(*ast.FuncLit)(nil),
	}

	targetNamesToSubstitute := []string{
		"ioutil.TempDir",
		"os.MkdirTemp",
	}

	inspect.Preorder(nodeFilter, func(n ast.Node) {
		switch n := n.(type) {
		case *ast.FuncDecl:
			checkFuncDecl(pass, n, pass.Fset.File(n.Pos()).Name(), targetNamesToSubstitute...)
		case *ast.FuncLit:
			checkFuncLit(pass, n, pass.Fset.File(n.Pos()).Name(), targetNamesToSubstitute...)
		}
	})

	return nil, nil
}

func checkFuncDecl(pass *analysis.Pass,
	f *ast.FuncDecl,
	fileName string,
	targetNamesToSubstitute ...string,
) {
	argName, ok := targetRunner(f.Type.Params.List, fileName)
	if !ok {
		return
	}
	checkStmts(pass, f.Body.List, f.Name.Name, argName, targetNamesToSubstitute...)
}

func checkFuncLit(pass *analysis.Pass,
	f *ast.FuncLit,
	fileName string,
	targetNamesToSubstitute ...string,
) {
	argName, ok := targetRunner(f.Type.Params.List, fileName)
	if !ok {
		return
	}
	checkStmts(pass, f.Body.List, "anonymous function", argName, targetNamesToSubstitute...)
}

func checkStmts(pass *analysis.Pass,
	stmts []ast.Stmt,
	funcName, argName string,
	targetNamesToSubstitute ...string,
) {
	for _, stmt := range stmts {
		switch stmt := stmt.(type) {
		case *ast.ExprStmt:
			if !checkExprStmt(pass, stmt, funcName, argName, targetNamesToSubstitute...) {
				continue
			}
		case *ast.IfStmt:
			if !checkIfStmt(pass, stmt, funcName, argName, targetNamesToSubstitute...) {
				continue
			}
		case *ast.AssignStmt:
			if !checkAssignStmt(pass, stmt, funcName, argName, targetNamesToSubstitute...) {
				continue
			}
		}
	}
}

func checkExprStmt(pass *analysis.Pass,
	stmt *ast.ExprStmt,
	funcName, argName string,
	targetNamesToSubstitute ...string,
) bool {
	callExpr, ok := stmt.X.(*ast.CallExpr)
	if !ok {
		return false
	}
	fun, ok := callExpr.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	x, ok := fun.X.(*ast.Ident)
	if !ok {
		return false
	}

	checkTargetNames(pass, stmt, funcName, argName, fun, x, targetNamesToSubstitute...)

	return true
}

func checkIfStmt(pass *analysis.Pass,
	stmt *ast.IfStmt, funcName,
	argName string,
	targetNamesToSubstitute ...string,
) bool {
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

	checkTargetNames(pass, stmt, funcName, argName, fun, x, targetNamesToSubstitute...)

	return true
}

func checkAssignStmt(pass *analysis.Pass,
	stmt *ast.AssignStmt,
	funcName, argName string,
	targetNamesToSubstitute ...string,
) bool {
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

	checkTargetNames(pass, stmt, funcName, argName, fun, x, targetNamesToSubstitute...)

	return true
}

func checkTargetNames(pass *analysis.Pass,
	stmt interface{ Pos() token.Pos },
	funcName, argName string,
	fun *ast.SelectorExpr,
	x *ast.Ident,
	targetNamesToSubstitute ...string,
) {
	targetName := x.Name + "." + fun.Sel.Name

	for _, toSubstitute := range targetNamesToSubstitute {
		if targetName == toSubstitute {
			if argName == "" {
				argName = "testing"
			}
			pass.Reportf(stmt.Pos(), "%s() can be replaced by `%s.TempDir()` in %s", toSubstitute, argName, funcName)

			return
		}
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
