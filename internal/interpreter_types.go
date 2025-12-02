package internal

import (
	"symbolic-execution-course/internal/symbolic"

	"golang.org/x/tools/go/ssa"
)

type Interpreter struct {
	CallStack     []CallStackFrame
	Analyser      *Analyser
	PathCondition symbolic.SymbolicExpression
	Heap          HeapInterface

	currentBlock *ssa.BasicBlock
	instrIndex   int

	loopCounters  map[string]int
	maxLoopUnroll int
	visitedBlocks map[string]bool

	prevBlock       *ssa.BasicBlock
	blockVisitCount map[string]int
}

type CallStackFrame struct {
	Function    *ssa.Function
	LocalMemory map[string]symbolic.SymbolicExpression
	ReturnValue symbolic.SymbolicExpression
}

type HeapInterface interface {
	Allocate(exprType symbolic.ExpressionType) *symbolic.Ref
	AssignField(ref *symbolic.Ref, fieldIndex int, value symbolic.SymbolicExpression)
	GetFieldValue(ref *symbolic.Ref, fieldIndex int) symbolic.SymbolicExpression
	AssignToArray(ref *symbolic.Ref, index int, value symbolic.SymbolicExpression)
	GetFromArray(ref *symbolic.Ref, index int) symbolic.SymbolicExpression
}
