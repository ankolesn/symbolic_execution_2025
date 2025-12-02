package internal

import (
	"fmt"
	"go/constant"
	"strconv"
	"strings"

	"symbolic-execution-course/internal/symbolic"

	"golang.org/x/tools/go/ssa"
)

const maxTotalUnrolls = 100
const maxLoopUnroll = 10 // Константа для максимального развертывания одного цикла

func (interpreter *Interpreter) GetCurrentFrame() *CallStackFrame {
	if len(interpreter.CallStack) == 0 {
		return nil
	}
	return &interpreter.CallStack[len(interpreter.CallStack)-1]
}

func (interpreter *Interpreter) IsFinished() bool {
	return interpreter.currentBlock == nil ||
		len(interpreter.CallStack) == 0 ||
		(interpreter.currentBlock != nil && interpreter.instrIndex >= len(interpreter.currentBlock.Instrs))
}

func (interpreter *Interpreter) GetNextInstruction() ssa.Instruction {
	if interpreter.IsFinished() {
		return nil
	}

	if interpreter.instrIndex < len(interpreter.currentBlock.Instrs) {
		return interpreter.currentBlock.Instrs[interpreter.instrIndex]
	}
	return nil
}

func (interpreter *Interpreter) initLoopSupport() {
	if interpreter.loopCounters == nil {
		interpreter.loopCounters = make(map[string]int)
	}
	if interpreter.visitedBlocks == nil {
		interpreter.visitedBlocks = make(map[string]bool)
	}
	if interpreter.maxLoopUnroll == 0 {
		interpreter.maxLoopUnroll = maxLoopUnroll
	}
	if interpreter.blockVisitCount == nil {
		interpreter.blockVisitCount = make(map[string]int)
	}
}

func (interpreter *Interpreter) Copy() *Interpreter {
	newInterpreter := &Interpreter{
		CallStack:       make([]CallStackFrame, len(interpreter.CallStack)),
		Analyser:        interpreter.Analyser,
		PathCondition:   interpreter.PathCondition,
		Heap:            interpreter.Heap,
		currentBlock:    interpreter.currentBlock,
		instrIndex:      interpreter.instrIndex,
		loopCounters:    make(map[string]int),
		maxLoopUnroll:   interpreter.maxLoopUnroll,
		visitedBlocks:   make(map[string]bool),
		blockVisitCount: make(map[string]int),
		prevBlock:       interpreter.prevBlock,
	}

	for k, v := range interpreter.loopCounters {
		newInterpreter.loopCounters[k] = v
	}

	for k, v := range interpreter.visitedBlocks {
		newInterpreter.visitedBlocks[k] = v
	}

	for k, v := range interpreter.blockVisitCount {
		newInterpreter.blockVisitCount[k] = v
	}

	for i, frame := range interpreter.CallStack {
		newFrame := CallStackFrame{
			Function:    frame.Function,
			LocalMemory: make(map[string]symbolic.SymbolicExpression),
			ReturnValue: frame.ReturnValue,
		}

		for k, v := range frame.LocalMemory {
			newFrame.LocalMemory[k] = v
		}

		newInterpreter.CallStack[i] = newFrame
	}

	return newInterpreter
}

func (interpreter *Interpreter) InterpretDynamically(element ssa.Instruction) []*Interpreter {
	interpreter.initLoopSupport()

	switch instr := element.(type) {
	case *ssa.Return:
		return interpreter.interpretReturn(instr)
	case *ssa.If:
		return interpreter.interpretIf(instr)
	case *ssa.Jump:
		return interpreter.interpretJump(instr)
	case *ssa.UnOp:
		return interpreter.interpretUnOp(instr)
	case *ssa.BinOp:
		return interpreter.interpretBinOp(instr)
	case *ssa.Store:
		return interpreter.interpretStore(instr)
	case *ssa.Alloc:
		return interpreter.interpretAlloc(instr)
	case *ssa.Phi:
		return interpreter.interpretPhi(instr)
	case *ssa.ChangeType:
		interpreter.instrIndex++
		return []*Interpreter{interpreter}
	case *ssa.Convert:
		return interpreter.interpretConvert(instr)
	case *ssa.Call:
		return interpreter.interpretCall(instr)
	case *ssa.MakeInterface:
		return interpreter.interpretMakeInterface(instr)
	case *ssa.FieldAddr:
		return interpreter.interpretFieldAddr(instr)
	case *ssa.Field:
		return interpreter.interpretField(instr)
	case *ssa.IndexAddr:
		return interpreter.interpretIndexAddr(instr)
	case *ssa.Index:
		return interpreter.interpretIndex(instr)
	default:
		if unop, ok := element.(*ssa.UnOp); ok && unop.Op.String() == "Load" {
			return interpreter.interpretLoad(unop)
		}
		interpreter.instrIndex++
		return []*Interpreter{interpreter}
	}
}

func (interpreter *Interpreter) ResolveExpression(value ssa.Value) symbolic.SymbolicExpression {
	if value == nil {
		return symbolic.NewIntConstant(0)
	}

	if value.Name() != "" {
		frame := interpreter.GetCurrentFrame()
		if frame != nil {
			if expr, ok := frame.LocalMemory[value.Name()]; ok {
				return expr
			}
		}
	}

	switch v := value.(type) {
	case *ssa.Const:
		return interpreter.resolveConst(v)
	case *ssa.UnOp:
		if v.Op.String() == "Load" {
			return interpreter.resolveLoad(v)
		}
		return interpreter.resolveUnOp(v)
	case *ssa.BinOp:
		return interpreter.resolveBinOp(v)
	case *ssa.Parameter:
		return interpreter.resolveParameter(v)
	case *ssa.Alloc:
		return interpreter.resolveAlloc(v)
	case *ssa.Phi:
		return interpreter.resolvePhi(v)
	case *ssa.Call:
		return interpreter.resolveCall(v)
	case *ssa.ChangeType:
		return interpreter.ResolveExpression(v.X)
	case *ssa.Convert:
		return interpreter.ResolveExpression(v.X)
	case *ssa.MakeInterface:
		return interpreter.ResolveExpression(v.X)
	case *ssa.FieldAddr:
		return interpreter.resolveFieldAddr(v)
	case *ssa.Field:
		return interpreter.resolveField(v)
	case *ssa.IndexAddr:
		return interpreter.resolveIndexAddr(v)
	case *ssa.Index:
		return interpreter.resolveIndex(v)
	default:
		if v != nil && v.Name() != "" {
			var exprType symbolic.ExpressionType
			typeStr := v.Type().String()
			if strings.Contains(typeStr, "int") {
				exprType = symbolic.IntType
			} else if typeStr == "bool" {
				exprType = symbolic.BoolType
			} else {
				exprType = symbolic.IntType
			}
			return symbolic.NewSymbolicVariable(v.Name(), exprType)
		}
		return symbolic.NewIntConstant(0)
	}
}

func (interpreter *Interpreter) interpretReturn(instr *ssa.Return) []*Interpreter {
	frame := interpreter.GetCurrentFrame()

	if len(instr.Results) > 0 {
		result := interpreter.ResolveExpression(instr.Results[0])
		frame.ReturnValue = result
	}

	interpreter.currentBlock = nil
	return []*Interpreter{interpreter}
}

func (interpreter *Interpreter) interpretIf(instr *ssa.If) []*Interpreter {
	condExpr := interpreter.ResolveExpression(instr.Cond)

	trueInterpreter := interpreter.Copy()
	falseInterpreter := interpreter.Copy()

	notCond := symbolic.NewUnaryOperation(condExpr, symbolic.UNARY_NOT)

	trueInterpreter.PathCondition = symbolic.NewLogicalOperation(
		[]symbolic.SymbolicExpression{interpreter.PathCondition, condExpr},
		symbolic.AND,
	)

	falseInterpreter.PathCondition = symbolic.NewLogicalOperation(
		[]symbolic.SymbolicExpression{interpreter.PathCondition, notCond},
		symbolic.AND,
	)

	trueInterpreter.prevBlock = interpreter.currentBlock
	falseInterpreter.prevBlock = interpreter.currentBlock

	if len(instr.Block().Succs) >= 2 {
		trueInterpreter.currentBlock = instr.Block().Succs[0]
		trueInterpreter.instrIndex = 0

		falseInterpreter.currentBlock = instr.Block().Succs[1]
		falseInterpreter.instrIndex = 0
	} else {
		trueInterpreter.currentBlock = nil
		falseInterpreter.currentBlock = nil
	}

	return []*Interpreter{trueInterpreter, falseInterpreter}
}

func (interpreter *Interpreter) interpretJump(instr *ssa.Jump) []*Interpreter {
	if len(instr.Block().Succs) > 0 {
		nextBlock := instr.Block().Succs[0]

		interpreter.prevBlock = interpreter.currentBlock

		blockKey := fmt.Sprintf("%p", nextBlock)
		visitCount := interpreter.blockVisitCount[blockKey]

		if visitCount >= interpreter.maxLoopUnroll {
			exitBlock := interpreter.findLoopExit(nextBlock)
			if exitBlock != nil {
				interpreter.currentBlock = exitBlock
				interpreter.instrIndex = 0
			} else {
				interpreter.currentBlock = nil
			}
			return []*Interpreter{interpreter}
		}

		interpreter.blockVisitCount[blockKey] = visitCount + 1

		if interpreter.totalUnrolls() >= maxTotalUnrolls {
			exitBlock := interpreter.findLoopExit(nextBlock)
			if exitBlock != nil {
				interpreter.currentBlock = exitBlock
				interpreter.instrIndex = 0
			} else {
				interpreter.currentBlock = nil
			}
			return []*Interpreter{interpreter}
		}

		interpreter.currentBlock = nextBlock
		interpreter.instrIndex = 0
	} else {
		interpreter.currentBlock = nil
	}
	return []*Interpreter{interpreter}
}

func (interpreter *Interpreter) handleLoop(loopHeader *ssa.BasicBlock) []*Interpreter {
	return interpreter.exitLoop(loopHeader)
}

func (interpreter *Interpreter) totalUnrolls() int {
	total := 0
	for _, count := range interpreter.blockVisitCount {
		total += count
	}
	return total
}

func (interpreter *Interpreter) findLoopExit(loopHeader *ssa.BasicBlock) *ssa.BasicBlock {
	//ищем блок, который не является частью цикла
	//ищем блок с return или блок, который ведет к выходу

	visited := make(map[*ssa.BasicBlock]bool)
	var queue []*ssa.BasicBlock

	//начинаем с преемников заголовка цикла
	for _, succ := range loopHeader.Succs {
		if succ != nil && succ != loopHeader {
			queue = append(queue, succ)
		}
	}

	for len(queue) > 0 {
		block := queue[0]
		queue = queue[1:]

		if visited[block] {
			continue
		}
		visited[block] = true

		//проверяем, является ли этот блок выходом
		//если блок содержит кeturn - это выход
		for _, instr := range block.Instrs {
			if _, ok := instr.(*ssa.Return); ok {
				return block
			}
		}

		//если блок не ведет обратно в заголовок цикла
		isPartOfLoop := false
		for _, succ := range block.Succs {
			if succ == loopHeader {
				isPartOfLoop = true
				break
			}
		}

		if !isPartOfLoop {
			return block
		}

		//добавляем преемников для дальнейшего поиска
		for _, succ := range block.Succs {
			if succ != nil && !visited[succ] {
				queue = append(queue, succ)
			}
		}
	}

	return nil
}

func (interpreter *Interpreter) exitLoop(loopHeader *ssa.BasicBlock) []*Interpreter {
	exitInterpreter := interpreter.Copy()
	exitBlock := interpreter.findLoopExit(loopHeader)
	if exitBlock != nil {
		exitInterpreter.currentBlock = exitBlock
		exitInterpreter.instrIndex = 0
		exitInterpreter.prevBlock = interpreter.currentBlock
	} else {
		exitInterpreter.currentBlock = nil
	}

	return []*Interpreter{exitInterpreter}
}

func (interpreter *Interpreter) interpretUnOp(instr *ssa.UnOp) []*Interpreter {
	operand := interpreter.ResolveExpression(instr.X)

	var unaryOp symbolic.UnaryOperator

	opStr := instr.Op.String()
	switch opStr {
	case "-":
		unaryOp = symbolic.UNARY_MINUS
	case "!":
		unaryOp = symbolic.UNARY_NOT
	case "^":
		interpreter.instrIndex++
		return []*Interpreter{interpreter}
	default:
		interpreter.instrIndex++
		return []*Interpreter{interpreter}
	}

	result := symbolic.NewUnaryOperation(operand, unaryOp)

	frame := interpreter.GetCurrentFrame()
	if frame != nil && instr.Name() != "" {
		frame.LocalMemory[instr.Name()] = result
	}

	interpreter.instrIndex++
	return []*Interpreter{interpreter}
}

func (interpreter *Interpreter) interpretBinOp(instr *ssa.BinOp) []*Interpreter {
	left := interpreter.ResolveExpression(instr.X)
	right := interpreter.ResolveExpression(instr.Y)

	var binOp symbolic.BinaryOperator

	opStr := instr.Op.String()

	opStr = strings.Trim(opStr, "\"'")

	isComparison := false

	// Преобразуем в enum
	switch opStr {
	case "+":
		binOp = symbolic.ADD
	case "-":
		binOp = symbolic.SUB
	case "*":
		binOp = symbolic.MUL
	case "/":
		binOp = symbolic.DIV
	case "%":
		binOp = symbolic.MOD
	case "==":
		binOp = symbolic.EQ
		isComparison = true
	case "!=":
		binOp = symbolic.NE
		isComparison = true
	case "<":
		binOp = symbolic.LT
		isComparison = true
	case "<=":
		binOp = symbolic.LE
		isComparison = true
	case ">":
		binOp = symbolic.GT
		isComparison = true
	case ">=":
		binOp = symbolic.GE
		isComparison = true
	case "&", "|", "^", "<<", ">>", "&^":
		interpreter.instrIndex++
		return []*Interpreter{interpreter}
	case "&&":
		result := symbolic.NewLogicalOperation([]symbolic.SymbolicExpression{left, right}, symbolic.AND)

		frame := interpreter.GetCurrentFrame()
		if frame != nil && instr.Name() != "" {
			frame.LocalMemory[instr.Name()] = result
		}

		interpreter.instrIndex++
		return []*Interpreter{interpreter}
	case "||":
		result := symbolic.NewLogicalOperation([]symbolic.SymbolicExpression{left, right}, symbolic.OR)

		frame := interpreter.GetCurrentFrame()
		if frame != nil && instr.Name() != "" {
			frame.LocalMemory[instr.Name()] = result
		}

		interpreter.instrIndex++
		return []*Interpreter{interpreter}
	default:
		interpreter.instrIndex++
		return []*Interpreter{interpreter}
	}

	var result symbolic.SymbolicExpression

	if isComparison {
		result = symbolic.NewBinaryOperation(left, right, binOp)
	} else {
		result = symbolic.NewBinaryOperation(left, right, binOp)
	}

	result = simplifyExpression(result)

	frame := interpreter.GetCurrentFrame()
	if frame != nil && instr.Name() != "" {
		frame.LocalMemory[instr.Name()] = result
	}

	interpreter.instrIndex++
	return []*Interpreter{interpreter}
}

func (interpreter *Interpreter) interpretAlloc(instr *ssa.Alloc) []*Interpreter {
	var exprType symbolic.ExpressionType
	typeStr := instr.Type().String()

	if strings.Contains(typeStr, "int") {
		exprType = symbolic.IntType
	} else if strings.Contains(typeStr, "struct") {
		exprType = symbolic.StructType
	} else if strings.Contains(typeStr, "[") && strings.Contains(typeStr, "]") {
		exprType = symbolic.ArrayType
	} else {
		exprType = symbolic.RefType
	}

	ref := interpreter.Heap.Allocate(exprType)

	frame := interpreter.GetCurrentFrame()
	if frame != nil && instr.Name() != "" {
		frame.LocalMemory[instr.Name()] = ref
	}

	interpreter.instrIndex++
	return []*Interpreter{interpreter}
}

func (interpreter *Interpreter) interpretConvert(instr *ssa.Convert) []*Interpreter {
	operand := interpreter.ResolveExpression(instr.X)

	frame := interpreter.GetCurrentFrame()
	if frame != nil && instr.Name() != "" {
		frame.LocalMemory[instr.Name()] = operand
	}

	interpreter.instrIndex++
	return []*Interpreter{interpreter}
}

func (interpreter *Interpreter) interpretStore(instr *ssa.Store) []*Interpreter {
	addr := interpreter.ResolveExpression(instr.Addr)
	value := interpreter.ResolveExpression(instr.Val)

	if fieldAddr, ok := addr.(*symbolic.FieldAddr); ok {
		interpreter.Heap.AssignField(fieldAddr.Ref, fieldAddr.FieldIndex, value)
	} else if indexAddr, ok := addr.(*symbolic.IndexAddr); ok {
		interpreter.Heap.AssignToArray(indexAddr.Ref, indexAddr.Index, value)
	} else if ref, ok := addr.(*symbolic.Ref); ok {
		interpreter.Heap.AssignField(ref, 0, value)
	}

	interpreter.instrIndex++
	return []*Interpreter{interpreter}
}

func (interpreter *Interpreter) interpretPhi(instr *ssa.Phi) []*Interpreter {
	//для PHI-функций выбираем значение в зависимости от того, из какого блока пришли
	frame := interpreter.GetCurrentFrame()
	if frame == nil {
		interpreter.instrIndex++
		return []*Interpreter{interpreter}
	}

	var result symbolic.SymbolicExpression

	if interpreter.prevBlock != nil {
		for i, pred := range instr.Block().Preds {
			if pred == interpreter.prevBlock && i < len(instr.Edges) {
				result = interpreter.ResolveExpression(instr.Edges[i])
				break
			}
		}
	}

	if result == nil && len(instr.Edges) > 0 {
		result = interpreter.ResolveExpression(instr.Edges[0])
	}

	if result == nil {
		result = symbolic.NewIntConstant(0)
	}

	result = simplifyExpression(result)

	if frame != nil && instr.Name() != "" {
		frame.LocalMemory[instr.Name()] = result
	}

	interpreter.instrIndex++
	return []*Interpreter{interpreter}
}

func (interpreter *Interpreter) interpretCall(instr *ssa.Call) []*Interpreter {
	frame := interpreter.GetCurrentFrame()

	args := make([]symbolic.SymbolicExpression, len(instr.Call.Args))
	for i, arg := range instr.Call.Args {
		args[i] = interpreter.ResolveExpression(arg)
	}

	funcName := "call_result"
	if instr.Call.Value != nil && instr.Call.Value.Name() != "" {
		funcName = instr.Call.Value.Name()
	}

	result := symbolic.NewSymbolicVariable(funcName, symbolic.IntType)

	if frame != nil && instr.Name() != "" {
		frame.LocalMemory[instr.Name()] = result
	}

	interpreter.instrIndex++
	return []*Interpreter{interpreter}
}

func (interpreter *Interpreter) interpretMakeInterface(instr *ssa.MakeInterface) []*Interpreter {
	value := interpreter.ResolveExpression(instr.X)

	frame := interpreter.GetCurrentFrame()
	if frame != nil && instr.Name() != "" {
		frame.LocalMemory[instr.Name()] = value
	}

	interpreter.instrIndex++
	return []*Interpreter{interpreter}
}

func (interpreter *Interpreter) interpretFieldAddr(instr *ssa.FieldAddr) []*Interpreter {
	base := interpreter.ResolveExpression(instr.X)

	var result symbolic.SymbolicExpression

	if ref, ok := base.(*symbolic.Ref); ok {
		fieldIndex := instr.Field
		result = symbolic.NewFieldAddr(ref, fieldIndex)
	} else {
		result = symbolic.NewSymbolicVariable(instr.Name(), symbolic.RefType)
	}

	frame := interpreter.GetCurrentFrame()
	if frame != nil && instr.Name() != "" {
		frame.LocalMemory[instr.Name()] = result
	}

	interpreter.instrIndex++
	return []*Interpreter{interpreter}
}

func (interpreter *Interpreter) interpretField(instr *ssa.Field) []*Interpreter {
	base := interpreter.ResolveExpression(instr.X)

	var result symbolic.SymbolicExpression

	if ref, ok := base.(*symbolic.Ref); ok {
		fieldIndex := instr.Field
		result = interpreter.Heap.GetFieldValue(ref, fieldIndex)
	} else {
		result = symbolic.NewIntConstant(0)
	}

	frame := interpreter.GetCurrentFrame()
	if frame != nil && instr.Name() != "" {
		frame.LocalMemory[instr.Name()] = result
	}

	interpreter.instrIndex++
	return []*Interpreter{interpreter}
}

func (interpreter *Interpreter) interpretIndexAddr(instr *ssa.IndexAddr) []*Interpreter {
	base := interpreter.ResolveExpression(instr.X)
	index := interpreter.ResolveExpression(instr.Index)

	var result symbolic.SymbolicExpression

	if ref, ok := base.(*symbolic.Ref); ok {
		if indexConst, ok := index.(*symbolic.IntConstant); ok {
			result = symbolic.NewIndexAddr(ref, int(indexConst.Value))
		} else {
			result = symbolic.NewIndexAddr(ref, 0)
		}
	} else {
		result = symbolic.NewSymbolicVariable(instr.Name(), symbolic.RefType)
	}

	frame := interpreter.GetCurrentFrame()
	if frame != nil && instr.Name() != "" {
		frame.LocalMemory[instr.Name()] = result
	}

	interpreter.instrIndex++
	return []*Interpreter{interpreter}
}

func (interpreter *Interpreter) interpretIndex(instr *ssa.Index) []*Interpreter {
	base := interpreter.ResolveExpression(instr.X)
	index := interpreter.ResolveExpression(instr.Index)

	var result symbolic.SymbolicExpression

	if ref, ok := base.(*symbolic.Ref); ok {
		if indexConst, ok := index.(*symbolic.IntConstant); ok {
			result = interpreter.Heap.GetFromArray(ref, int(indexConst.Value))
		} else {
			result = symbolic.NewIntConstant(0)
		}
	} else {
		result = symbolic.NewIntConstant(0)
	}

	frame := interpreter.GetCurrentFrame()
	if frame != nil && instr.Name() != "" {
		frame.LocalMemory[instr.Name()] = result
	}

	interpreter.instrIndex++
	return []*Interpreter{interpreter}
}

func (interpreter *Interpreter) interpretLoad(instr *ssa.UnOp) []*Interpreter {
	addr := interpreter.ResolveExpression(instr.X)

	var result symbolic.SymbolicExpression

	switch a := addr.(type) {
	case *symbolic.Ref:
		result = interpreter.Heap.GetFieldValue(a, 0)
	case *symbolic.FieldAddr:
		result = interpreter.Heap.GetFieldValue(a.Ref, a.FieldIndex)
	case *symbolic.IndexAddr:
		result = interpreter.Heap.GetFromArray(a.Ref, a.Index)
	default:
		result = symbolic.NewIntConstant(0)
	}

	result = simplifyExpression(result)

	frame := interpreter.GetCurrentFrame()
	if frame != nil && instr.Name() != "" {
		frame.LocalMemory[instr.Name()] = result
	}

	interpreter.instrIndex++
	return []*Interpreter{interpreter}
}

func (interpreter *Interpreter) resolveLoad(l *ssa.UnOp) symbolic.SymbolicExpression {
	if l.Name() != "" {
		frame := interpreter.GetCurrentFrame()
		if frame != nil {
			if expr, ok := frame.LocalMemory[l.Name()]; ok {
				return expr
			}
		}
	}

	addr := interpreter.ResolveExpression(l.X)

	var result symbolic.SymbolicExpression
	switch a := addr.(type) {
	case *symbolic.Ref:
		result = interpreter.Heap.GetFieldValue(a, 0)
	case *symbolic.FieldAddr:
		result = interpreter.Heap.GetFieldValue(a.Ref, a.FieldIndex)
	case *symbolic.IndexAddr:
		result = interpreter.Heap.GetFromArray(a.Ref, a.Index)
	default:
		result = symbolic.NewIntConstant(0)
	}

	return simplifyExpression(result)
}

func (interpreter *Interpreter) resolveConst(c *ssa.Const) symbolic.SymbolicExpression {
	if c.IsNil() {
		return symbolic.NewIntConstant(0)
	}

	val := c.Value
	if val == nil {
		return symbolic.NewIntConstant(0)
	}

	switch val.Kind() {
	case constant.Int:
		if intVal, ok := constant.Int64Val(val); ok {
			return symbolic.NewIntConstant(intVal)
		}
	case constant.Bool:
		boolVal := constant.BoolVal(val)
		return symbolic.NewBoolConstant(boolVal)
	case constant.String:
		return symbolic.NewIntConstant(0)
	case constant.Float:
		if floatStr := val.String(); floatStr != "" {
			if f, err := strconv.ParseFloat(floatStr, 64); err == nil {
				return symbolic.NewIntConstant(int64(f))
			}
		}
		return symbolic.NewIntConstant(0)
	}

	return symbolic.NewIntConstant(0)
}

func (interpreter *Interpreter) resolveUnOp(u *ssa.UnOp) symbolic.SymbolicExpression {
	if u.Name() != "" {
		frame := interpreter.GetCurrentFrame()
		if frame != nil {
			if expr, ok := frame.LocalMemory[u.Name()]; ok {
				return expr
			}
		}
	}

	operand := interpreter.ResolveExpression(u.X)

	var unaryOp symbolic.UnaryOperator
	opStr := u.Op.String()

	switch opStr {
	case "-":
		unaryOp = symbolic.UNARY_MINUS
	case "!":
		unaryOp = symbolic.UNARY_NOT
	default:
		return operand
	}

	result := symbolic.NewUnaryOperation(operand, unaryOp)
	return simplifyExpression(result)
}

func (interpreter *Interpreter) resolveBinOp(b *ssa.BinOp) symbolic.SymbolicExpression {
	if b.Name() != "" {
		frame := interpreter.GetCurrentFrame()
		if frame != nil {
			if expr, ok := frame.LocalMemory[b.Name()]; ok {
				return expr
			}
		}
	}

	left := interpreter.ResolveExpression(b.X)
	right := interpreter.ResolveExpression(b.Y)

	var binOp symbolic.BinaryOperator
	opStr := b.Op.String()
	opStr = strings.Trim(opStr, "\"'")

	switch opStr {
	case "+":
		binOp = symbolic.ADD
	case "-":
		binOp = symbolic.SUB
	case "*":
		binOp = symbolic.MUL
	case "/":
		binOp = symbolic.DIV
	case "%":
		binOp = symbolic.MOD
	case "==":
		binOp = symbolic.EQ
	case "!=":
		binOp = symbolic.NE
	case "<":
		binOp = symbolic.LT
	case "<=":
		binOp = symbolic.LE
	case ">":
		binOp = symbolic.GT
	case ">=":
		binOp = symbolic.GE
	case "&&":
		result := symbolic.NewLogicalOperation([]symbolic.SymbolicExpression{left, right}, symbolic.AND)
		return simplifyExpression(result)
	case "||":
		result := symbolic.NewLogicalOperation([]symbolic.SymbolicExpression{left, right}, symbolic.OR)
		return simplifyExpression(result)
	default:
		return left
	}

	result := symbolic.NewBinaryOperation(left, right, binOp)
	return simplifyExpression(result)
}

func (interpreter *Interpreter) resolveParameter(p *ssa.Parameter) symbolic.SymbolicExpression {
	frame := interpreter.GetCurrentFrame()
	if frame != nil {
		if val, ok := frame.LocalMemory[p.Name()]; ok {
			return val
		}
	}

	var exprType symbolic.ExpressionType
	typeStr := p.Type().String()
	if strings.Contains(typeStr, "int") {
		exprType = symbolic.IntType
	} else if typeStr == "bool" {
		exprType = symbolic.BoolType
	} else {
		exprType = symbolic.IntType
	}

	return symbolic.NewSymbolicVariable(p.Name(), exprType)
}

func (interpreter *Interpreter) resolveAlloc(a *ssa.Alloc) symbolic.SymbolicExpression {
	frame := interpreter.GetCurrentFrame()
	if frame != nil && a.Name() != "" {
		if val, ok := frame.LocalMemory[a.Name()]; ok {
			return val
		}
	}

	var exprType symbolic.ExpressionType
	typeStr := a.Type().String()

	if strings.Contains(typeStr, "int") {
		exprType = symbolic.IntType
	} else if strings.Contains(typeStr, "struct") {
		exprType = symbolic.StructType
	} else if strings.Contains(typeStr, "[") && strings.Contains(typeStr, "]") {
		exprType = symbolic.ArrayType
	} else {
		exprType = symbolic.RefType
	}

	return interpreter.Heap.Allocate(exprType)
}

func (interpreter *Interpreter) resolvePhi(phi *ssa.Phi) symbolic.SymbolicExpression {
	frame := interpreter.GetCurrentFrame()
	if frame != nil && phi.Name() != "" {
		if val, ok := frame.LocalMemory[phi.Name()]; ok {
			return val
		}
	}

	for _, edge := range phi.Edges {
		if edge != nil && edge.Name() != "" {
			if expr, ok := frame.LocalMemory[edge.Name()]; ok {
				return simplifyExpression(expr)
			}
		}
	}

	if len(phi.Edges) > 0 {
		result := interpreter.ResolveExpression(phi.Edges[0])
		return simplifyExpression(result)
	}

	return symbolic.NewIntConstant(0)
}

func (interpreter *Interpreter) resolveCall(c *ssa.Call) symbolic.SymbolicExpression {
	frame := interpreter.GetCurrentFrame()
	if frame != nil && c.Name() != "" {
		if val, ok := frame.LocalMemory[c.Name()]; ok {
			return val
		}
	}

	funcName := "call_result"
	if c.Call.Value != nil && c.Call.Value.Name() != "" {
		funcName = c.Call.Value.Name()
	}

	return symbolic.NewSymbolicVariable(funcName, symbolic.IntType)
}

func (interpreter *Interpreter) resolveFieldAddr(f *ssa.FieldAddr) symbolic.SymbolicExpression {
	if f.Name() != "" {
		frame := interpreter.GetCurrentFrame()
		if frame != nil {
			if expr, ok := frame.LocalMemory[f.Name()]; ok {
				return expr
			}
		}
	}

	base := interpreter.ResolveExpression(f.X)

	if ref, ok := base.(*symbolic.Ref); ok {
		return symbolic.NewFieldAddr(ref, f.Field)
	}

	return symbolic.NewSymbolicVariable(f.Name(), symbolic.RefType)
}

func (interpreter *Interpreter) resolveField(f *ssa.Field) symbolic.SymbolicExpression {
	if f.Name() != "" {
		frame := interpreter.GetCurrentFrame()
		if frame != nil {
			if expr, ok := frame.LocalMemory[f.Name()]; ok {
				return expr
			}
		}
	}

	base := interpreter.ResolveExpression(f.X)

	if ref, ok := base.(*symbolic.Ref); ok {
		result := interpreter.Heap.GetFieldValue(ref, f.Field)
		return simplifyExpression(result)
	}

	return symbolic.NewIntConstant(0)
}

func (interpreter *Interpreter) resolveIndexAddr(i *ssa.IndexAddr) symbolic.SymbolicExpression {
	if i.Name() != "" {
		frame := interpreter.GetCurrentFrame()
		if frame != nil {
			if expr, ok := frame.LocalMemory[i.Name()]; ok {
				return expr
			}
		}
	}

	base := interpreter.ResolveExpression(i.X)
	index := interpreter.ResolveExpression(i.Index)

	if ref, ok := base.(*symbolic.Ref); ok {
		if indexConst, ok := index.(*symbolic.IntConstant); ok {
			return symbolic.NewIndexAddr(ref, int(indexConst.Value))
		}
		return symbolic.NewIndexAddr(ref, 0)
	}

	return symbolic.NewSymbolicVariable(i.Name(), symbolic.RefType)
}

func (interpreter *Interpreter) resolveIndex(i *ssa.Index) symbolic.SymbolicExpression {
	if i.Name() != "" {
		frame := interpreter.GetCurrentFrame()
		if frame != nil {
			if expr, ok := frame.LocalMemory[i.Name()]; ok {
				return expr
			}
		}
	}

	base := interpreter.ResolveExpression(i.X)
	index := interpreter.ResolveExpression(i.Index)

	if ref, ok := base.(*symbolic.Ref); ok {
		if indexConst, ok := index.(*symbolic.IntConstant); ok {
			result := interpreter.Heap.GetFromArray(ref, int(indexConst.Value))
			return simplifyExpression(result)
		}
		return symbolic.NewIntConstant(0)
	}

	return symbolic.NewIntConstant(0)
}

func simplifyExpression(expr symbolic.SymbolicExpression) symbolic.SymbolicExpression {
	if expr == nil {
		return expr
	}

	switch e := expr.(type) {
	case *symbolic.BinaryOperation:
		left := simplifyExpression(e.Left)
		right := simplifyExpression(e.Right)

		if leftConst, ok := left.(*symbolic.IntConstant); ok {
			if rightConst, ok := right.(*symbolic.IntConstant); ok {
				switch e.Operator {
				case symbolic.ADD:
					return symbolic.NewIntConstant(leftConst.Value + rightConst.Value)
				case symbolic.SUB:
					return symbolic.NewIntConstant(leftConst.Value - rightConst.Value)
				case symbolic.MUL:
					return symbolic.NewIntConstant(leftConst.Value * rightConst.Value)
				case symbolic.DIV:
					if rightConst.Value != 0 {
						return symbolic.NewIntConstant(leftConst.Value / rightConst.Value)
					}
				case symbolic.MOD:
					if rightConst.Value != 0 {
						return symbolic.NewIntConstant(leftConst.Value % rightConst.Value)
					}
				}
			}
		}

		if e.Operator == symbolic.ADD {
			if leftConst, ok := left.(*symbolic.IntConstant); ok && leftConst.Value == 0 {
				return right
			}
			if rightConst, ok := right.(*symbolic.IntConstant); ok && rightConst.Value == 0 {
				return left
			}
		}

		if e.Operator == symbolic.MUL {
			if leftConst, ok := left.(*symbolic.IntConstant); ok && leftConst.Value == 0 {
				return symbolic.NewIntConstant(0)
			}
			if rightConst, ok := right.(*symbolic.IntConstant); ok && rightConst.Value == 0 {
				return symbolic.NewIntConstant(0)
			}
		}

		if e.Operator == symbolic.SUB {
			if rightConst, ok := right.(*symbolic.IntConstant); ok && rightConst.Value == 0 {
				return left
			}
		}

		if left != e.Left || right != e.Right {
			return symbolic.NewBinaryOperation(left, right, e.Operator)
		}
		return expr

	case *symbolic.UnaryOperation:
		operand := simplifyExpression(e.Operand)

		if operandConst, ok := operand.(*symbolic.IntConstant); ok {
			switch e.Operator {
			case symbolic.UNARY_MINUS:
				return symbolic.NewIntConstant(-operandConst.Value)
			case symbolic.UNARY_NOT:
				if operandConst.Value == 0 {
					return symbolic.NewBoolConstant(true)
				} else {
					return symbolic.NewBoolConstant(false)
				}
			}
		}

		if e.Operator == symbolic.UNARY_NOT {
			if nestedUnary, ok := operand.(*symbolic.UnaryOperation); ok && nestedUnary.Operator == symbolic.UNARY_NOT {
				return simplifyExpression(nestedUnary.Operand)
			}
		}

		if operand != e.Operand {
			return symbolic.NewUnaryOperation(operand, e.Operator)
		}
		return expr

	case *symbolic.LogicalOperation:
		simplifiedOperands := make([]symbolic.SymbolicExpression, len(e.Operands))
		changed := false

		for i, operand := range e.Operands {
			simplified := simplifyExpression(operand)
			simplifiedOperands[i] = simplified
			if simplified != operand {
				changed = true
			}
		}

		if changed {
			return symbolic.NewLogicalOperation(simplifiedOperands, e.Operator)
		}
		return expr

	default:
		return expr
	}
}

func (interpreter *Interpreter) String() string {
	result := fmt.Sprintf("Interpreter:\n")
	result += fmt.Sprintf("PathCondition: %s\n", interpreter.PathCondition.String())

	if len(interpreter.CallStack) > 0 {
		frame := interpreter.GetCurrentFrame()
		result += fmt.Sprintf("Current Frame:\n")
		result += fmt.Sprintf("Function: %s\n", frame.Function.Name())

		if len(frame.LocalMemory) > 0 {
			result += fmt.Sprintf("LocalMemory:\n")
			for k, v := range frame.LocalMemory {
				result += fmt.Sprintf("%s: %s\n", k, v.String())
			}
		}

		if frame.ReturnValue != nil {
			result += fmt.Sprintf("ReturnValue: %s\n", frame.ReturnValue.String())
		}
	}

	if interpreter.currentBlock != nil {
		result += fmt.Sprintf("CurrentBlock: %s\n", interpreter.currentBlock.String())
	}
	result += fmt.Sprintf("InstrIndex: %d\n", interpreter.instrIndex)
	result += fmt.Sprintf("TotalUnrolls: %d\n", interpreter.totalUnrolls())

	return result
}
