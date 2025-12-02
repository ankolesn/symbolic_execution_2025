package internal

import (
	"math/rand"
	"symbolic-execution-course/internal/symbolic"
	"symbolic-execution-course/internal/translator"
)

type PathSelector interface {
	CalculatePriority(interpreter Interpreter) int
}

type DfsPathSelector struct {
	counter int
}

func (dfs *DfsPathSelector) CalculatePriority(interpreter Interpreter) int {
	// DFS: чем больше counter, тем выше приоритет (глубже)
	dfs.counter++
	return dfs.counter
}

type BfsPathSelector struct {
	counter int
}

func (bfs *BfsPathSelector) CalculatePriority(interpreter Interpreter) int {
	// BFS: чем меньше counter, тем выше приоритет (шире)
	bfs.counter--
	return bfs.counter
}

type RandomPathSelector struct{}

func (random *RandomPathSelector) CalculatePriority(interpreter Interpreter) int {
	return rand.Int()
}

// приоритет по глубине пути
type DepthPathSelector struct{}

func (dps *DepthPathSelector) CalculatePriority(interpreter Interpreter) int {
	// считаем количество выполненных инструкций как меру глубины
	// чем больше инструкций выполнено, тем выше приоритет
	return interpreter.instrIndex + len(interpreter.CallStack)*1000
}

// приоритет по сложности условия пути
type ComplexityPathSelector struct {
	translator *translator.Z3Translator
}

func NewComplexityPathSelector(translator *translator.Z3Translator) *ComplexityPathSelector {
	return &ComplexityPathSelector{
		translator: translator,
	}
}

func (cps *ComplexityPathSelector) CalculatePriority(interpreter Interpreter) int {
	//Оцениваем сложность условия пути
	//Чем сложнее условие, тем выше приоритет
	//Упрощенная реализация: считаем количество операций в условии
	complexity := estimateComplexity(interpreter.PathCondition)
	return complexity
}

func estimateComplexity(expr symbolic.SymbolicExpression) int {
	switch e := expr.(type) {
	case *symbolic.SymbolicVariable:
		return 1
	case *symbolic.IntConstant, *symbolic.BoolConstant:
		return 1
	case *symbolic.BinaryOperation:
		return 1 + estimateComplexity(e.Left) + estimateComplexity(e.Right)
	case *symbolic.LogicalOperation:
		sum := 1
		for _, op := range e.Operands {
			sum += estimateComplexity(op)
		}
		return sum
	case *symbolic.UnaryOperation:
		return 1 + estimateComplexity(e.Operand)
	case *symbolic.Ref:
		return 1
	default:
		return 1
	}
}
