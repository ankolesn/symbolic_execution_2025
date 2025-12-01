package internal

import (
	"container/heap"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"log"

	"symbolic-execution-course/internal/memory"
	"symbolic-execution-course/internal/symbolic"
	"symbolic-execution-course/internal/translator"

	"golang.org/x/tools/go/ssa"
)

type Analyser struct {
	Package      *ssa.Package
	StatesQueue  PriorityQueue
	PathSelector PathSelector
	Results      []Interpreter
	Z3Translator translator.Translator
	maxSteps     int
	stepsCounter int
}

type Builder struct {
	fset *token.FileSet
}

func NewBuilder() *Builder {
	return &Builder{
		fset: token.NewFileSet(),
	}
}

func (b *Builder) ParseAndBuildSSA(source string, funcName string) (*ssa.Function, error) {
	f, err := parser.ParseFile(b.fset, "test.go", source, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("ошибка парсинга: %v", err)
	}

	conf := types.Config{
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

	pkg, err := conf.Check("main", b.fset, []*ast.File{f}, info)
	if err != nil {
		return nil, fmt.Errorf("ошибка проверки типов: %v", err)
	}

	prog := ssa.NewProgram(b.fset, ssa.SanityCheckFunctions)

	ssaPkg := prog.CreatePackage(pkg, []*ast.File{f}, info, true)

	ssaPkg.Build()

	for _, member := range ssaPkg.Members {
		if fn, ok := member.(*ssa.Function); ok && fn.Name() == funcName {
			return fn, nil
		}
	}

	return nil, fmt.Errorf("функция %s не найдена", funcName)
}

func (b *Builder) PrintFunctionInfo(fn *ssa.Function) {
	if fn == nil {
		println("Функция не найдена")
		return
	}

	fmt.Printf("Функция: %s\n", fn.Name())
	fmt.Printf("Количество блоков: %d\n", len(fn.Blocks))

	for i, param := range fn.Params {
		fmt.Printf("  Параметр %d: %s (тип: %s)\n", i, param.Name(), param.Type().String())
	}

	for i, block := range fn.Blocks {
		fmt.Printf("\nБлок %d:\n", i)
		for j, instr := range block.Instrs {
			fmt.Printf("    %d: %T: %s\n", j, instr, instr.String())
		}
	}
}

func Analyse(source string, functionName string) []Interpreter {
	return AnalyseWithOptions(source, functionName, &DfsPathSelector{}, 10000, true)
}

func AnalyseWithOptions(source string, functionName string, selector PathSelector, maxSteps int, verbose bool) []Interpreter {
	builder := NewBuilder()
	fn, err := builder.ParseAndBuildSSA(source, functionName)
	if err != nil {
		log.Printf("Ошибка построения SSA: %v", err)
		return nil
	}

	analyser := &Analyser{
		Package:      fn.Pkg,
		StatesQueue:  make(PriorityQueue, 0),
		PathSelector: selector,
		Results:      make([]Interpreter, 0),
		Z3Translator: translator.NewZ3Translator(),
		maxSteps:     maxSteps,
		stepsCounter: 0,
	}

	if verbose {
		fmt.Printf("=== Начало анализа функции %s ===\n", functionName)
		builder.PrintFunctionInfo(fn)
	}

	initialInterpreter := createInitialInterpreter(fn, analyser)

	heap.Init(&analyser.StatesQueue)

	heap.Push(&analyser.StatesQueue, &Item{
		value:    initialInterpreter,
		priority: analyser.PathSelector.CalculatePriority(initialInterpreter),
	})

	for analyser.StatesQueue.Len() > 0 && analyser.stepsCounter < analyser.maxSteps {
		item := heap.Pop(&analyser.StatesQueue).(*Item)
		interpreter := item.value
		interpreter.Analyser = analyser
		analyser.stepsCounter++

		if verbose {
			fmt.Printf("\n=== Шаг %d ===\n", analyser.stepsCounter)
			fmt.Printf("Состояний в очереди: %d\n", analyser.StatesQueue.Len())
			fmt.Printf("Условие пути: %s\n", interpreter.PathCondition.String())
		}

		if interpreter.isFinished() {
			if verbose {
				fmt.Printf("Состояние завершено\n")
			}
			analyser.Results = append(analyser.Results, interpreter)
			continue
		}

		nextInstruction := interpreter.getNextInstruction()
		if verbose && nextInstruction != nil {
			fmt.Printf("Инструкция: %T: %s\n", nextInstruction, nextInstruction.String())

			if ifInstr, ok := nextInstruction.(*ssa.If); ok {
				fmt.Printf("  Условие If: %T, имя: %s\n", ifInstr.Cond, ifInstr.Cond.Name())
				fmt.Printf("  Блоки преемники: %v\n", ifInstr.Block().Succs)
			}
		}
		if nextInstruction == nil {
			analyser.Results = append(analyser.Results, interpreter)
			continue
		}

		if verbose {
			fmt.Printf("Инструкция: %T: %s\n", nextInstruction, nextInstruction.String())
		}

		newStates := interpreter.interpretDynamically(nextInstruction)

		if verbose {
			fmt.Printf("Получено новых состояний: %d\n", len(newStates))
		}

		for _, newState := range newStates {
			newState.Analyser = analyser
			heap.Push(&analyser.StatesQueue, &Item{
				value:    newState,
				priority: analyser.PathSelector.CalculatePriority(newState),
			})
		}
	}

	if verbose {
		fmt.Printf("\n=== Результаты анализа ===\n")
		fmt.Printf("Всего шагов: %d\n", analyser.stepsCounter)
		fmt.Printf("Найдено завершенных состояний: %d\n", len(analyser.Results))

		for i, result := range analyser.Results {
			fmt.Printf("\nСостояние %d:\n", i)
			fmt.Printf("  Условие пути: %s\n", result.PathCondition.String())
			if frame := result.getCurrentFrame(); frame != nil && frame.ReturnValue != nil {
				fmt.Printf("  Возвращаемое значение: %s\n", frame.ReturnValue.String())
			}
		}
	}

	return analyser.Results
}

func createInitialInterpreter(fn *ssa.Function, analyser *Analyser) Interpreter {
	initialFrame := CallStackFrame{
		Function:    fn,
		LocalMemory: make(map[string]symbolic.SymbolicExpression),
		ReturnValue: nil,
	}

	for _, param := range fn.Params {
		switch param.Type().String() {
		case "int":
			initialFrame.LocalMemory[param.Name()] = symbolic.NewSymbolicVariable(param.Name(), symbolic.IntType)
		case "bool":
			initialFrame.LocalMemory[param.Name()] = symbolic.NewSymbolicVariable(param.Name(), symbolic.BoolType)
		default:
			initialFrame.LocalMemory[param.Name()] = symbolic.NewSymbolicVariable(param.Name(), symbolic.IntType)
		}
	}

	mem := memory.NewSymbolicMemory()

	return Interpreter{
		CallStack:     []CallStackFrame{initialFrame},
		Analyser:      analyser,
		PathCondition: symbolic.NewBoolConstant(true),
		Heap:          mem,
		currentBlock:  fn.Blocks[0],
		instrIndex:    0,
	}
}
