// Package ssa предоставляет функции для построения SSA представления
package ssa

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/ssa"
)

// Builder отвечает за построение SSA из исходного кода Go
type Builder struct {
	fset *token.FileSet
}

// NewBuilder создаёт новый экземпляр Builder
func NewBuilder() *Builder {
	return &Builder{
		fset: token.NewFileSet(),
	}
}

// ParseAndBuildSSA парсит исходный код Go и создаёт SSA представление
func (b *Builder) ParseAndBuildSSA(source string, funcName string) (*ssa.Function, error) {
	f, err := parser.ParseFile(b.fset, "homework1/main.go", source, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	files := []*ast.File{f}

	config := &types.Config{
		Importer: nil,
	}

	info := &types.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
		Implicits:  make(map[ast.Node]types.Object),
		Scopes:     make(map[ast.Node]*types.Scope),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
	}

	pkg, err := config.Check("homework1/main.go", b.fset, files, info)
	if err != nil {
		return nil, err
	}

	prog := ssa.NewProgram(b.fset, ssa.SanityCheckFunctions)
	
	ssaPkg := prog.CreatePackage(pkg, files, info, false)
	ssaPkg.Build()

	return ssaPkg.Func(funcName), nil
}

func (b *Builder) PrintFunctionInfo(fn *ssa.Function) {
	if fn == nil {
		println("Функция не найдена")
		return
	}

	println("Количество блоков:", len(fn.Blocks))
	
	for i, param := range fn.Params {
		println("  Параметр", i, ":", param.Name(), param.Type().String())
	}

	for i, block := range fn.Blocks {
		println("\nБлoк", i, ":")
		for j, instr := range block.Instrs {
			println("    ", j, ":", instr.String())
		}
	}
}