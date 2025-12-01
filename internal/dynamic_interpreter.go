package internal

import (
	"fmt"
	"go/constant"
	"go/token"

	"symbolic-execution-course/internal/memory"
	"symbolic-execution-course/internal/symbolic"

	"golang.org/x/tools/go/ssa"
)

type Interpreter struct {
	CallStack     []CallStackFrame
	Analyser      *Analyser
	PathCondition symbolic.SymbolicExpression
	Heap          memory.Memory
	currentBlock  *ssa.BasicBlock
	instrIndex    int
}

type CallStackFrame struct {
	Function    *ssa.Function
	LocalMemory map[string]symbolic.SymbolicExpression
	ReturnValue symbolic.SymbolicExpression
}

func (interpreter *Interpreter) getCurrentFrame() *CallStackFrame {
	if len(interpreter.CallStack) == 0 {
		return nil
	}
	return &interpreter.CallStack[len(interpreter.CallStack)-1]
}

func (interpreter *Interpreter) isFinished() bool {
	return interpreter.currentBlock == nil ||
		len(interpreter.CallStack) == 0 ||
		(interpreter.currentBlock != nil && interpreter.instrIndex >= len(interpreter.currentBlock.Instrs))
}

func (interpreter *Interpreter) getNextInstruction() ssa.Instruction {
	if interpreter.isFinished() {
		return nil
	}

	if interpreter.instrIndex < len(interpreter.currentBlock.Instrs) {
		return interpreter.currentBlock.Instrs[interpreter.instrIndex]
	}
	return nil
}

func (interpreter *Interpreter) copy() Interpreter {
	newInterpreter := Interpreter{
		CallStack:     make([]CallStackFrame, len(interpreter.CallStack)),
		Analyser:      interpreter.Analyser,
		PathCondition: interpreter.PathCondition,
		Heap:          interpreter.Heap,
		currentBlock:  interpreter.currentBlock,
		instrIndex:    interpreter.instrIndex,
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

func (interpreter *Interpreter) interpretDynamically(element ssa.Instruction) []Interpreter {
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
		interpreter.instrIndex++
		return []Interpreter{*interpreter}
	case *ssa.Alloc:
		return interpreter.interpretAlloc(instr)
	default:
		interpreter.instrIndex++
		return []Interpreter{*interpreter}
	}
}

// Метод resolveExpression
func (interpreter *Interpreter) resolveExpression(value ssa.Value) symbolic.SymbolicExpression {
	fmt.Printf("=== ОТЛАДКА resolveExpression ===\n")
	fmt.Printf("  Входное значение: %T, имя: '%s', строка: %s\n",
		value, value.Name(), value.String())

	if value.Name() != "" {
		frame := interpreter.getCurrentFrame()
		if frame != nil {
			if expr, ok := frame.LocalMemory[value.Name()]; ok {
				fmt.Printf("  Найдено в локальной памяти: %s (тип: %v)\n",
					expr.String(), expr.Type())
				return expr
			} else {
				fmt.Printf("  Не найдено в локальной памяти под именем '%s'\n", value.Name())
			}
		}
	}

	// Если нет в локальной памяти, разрешаем по типу
	switch v := value.(type) {
	case *ssa.Const:
		fmt.Printf("  Это Const\n")
		return interpreter.resolveConst(v)
	case *ssa.UnOp:
		fmt.Printf("  Это UnOp\n")
		return interpreter.resolveUnOp(v)
	case *ssa.BinOp:
		fmt.Printf("  Это BinOp: операция %v\n", v.Op)
		result := interpreter.resolveBinOp(v)
		fmt.Printf("  Результат BinOp: %s (тип: %v)\n", result.String(), result.Type())

		// Сохраняем в локальную память, если есть имя
		if v.Name() != "" {
			frame := interpreter.getCurrentFrame()
			if frame != nil {
				fmt.Printf("  Сохраняем в локальную память как '%s'\n", v.Name())
				frame.LocalMemory[v.Name()] = result
			}
		}
		return result
	case *ssa.Parameter:
		fmt.Printf("  Это Parameter\n")
		return interpreter.resolveParameter(v)
	case *ssa.Alloc:
		fmt.Printf("  Это Alloc\n")
		return interpreter.resolveAlloc(v)
	case *ssa.Phi:
		fmt.Printf("  Это Phi\n")
		return interpreter.resolvePhi(v)
	default:
		fmt.Printf("  Неизвестный тип: %T\n", v)
		// Для неизвестных значений создаем переменную
		if v != nil && v.Name() != "" {
			return symbolic.NewSymbolicVariable(v.Name(), symbolic.IntType)
		}
		return symbolic.NewIntConstant(0)
	}
}

func (interpreter *Interpreter) interpretReturn(instr *ssa.Return) []Interpreter {
	frame := interpreter.getCurrentFrame()

	if len(instr.Results) > 0 {
		result := interpreter.resolveExpression(instr.Results[0])
		frame.ReturnValue = result
	}

	interpreter.currentBlock = nil
	return []Interpreter{*interpreter}
}

func (interpreter *Interpreter) interpretIf(instr *ssa.If) []Interpreter {
	fmt.Printf("=== ОТЛАДКА interpretIf ===\n")
	fmt.Printf("instr.Cond тип: %T, значение: %s, имя: %s\n",
		instr.Cond, instr.Cond.String(), instr.Cond.Name())

	// Сначала попробуем разрешить как BinOp
	if binOp, ok := instr.Cond.(*ssa.BinOp); ok {
		fmt.Printf("Условие If - это BinOp: %s > %s\n",
			binOp.X.String(), binOp.Y.String())
		fmt.Printf("Операция BinOp: %v (10 = GTR)\n", binOp.Op)
	}

	condExpr := interpreter.resolveExpression(instr.Cond)

	fmt.Printf("condExpr после resolveExpression: тип=%v, значение=%s\n",
		condExpr.Type(), condExpr.String())

	// Убедимся, что condExpr - булев тип
	if condExpr.Type() != symbolic.BoolType {
		fmt.Printf("условие if не булево: тип %v, значение %s\n",
			condExpr.Type(), condExpr.String())

		// Попробуем найти значение в локальной памяти еще раз
		frame := interpreter.getCurrentFrame()
		if frame != nil && instr.Cond.Name() != "" {
			fmt.Printf("Ищем '%s' в локальной памяти: ", instr.Cond.Name())
			if val, ok := frame.LocalMemory[instr.Cond.Name()]; ok {
				fmt.Printf("найдено: %s (тип: %v)\n", val.String(), val.Type())
				condExpr = val
			} else {
				fmt.Printf("не найдено\n")
			}
		}

		if condExpr.Type() != symbolic.BoolType {
			panic(fmt.Sprintf("Условие if не булево: тип %v", condExpr.Type()))
		}
	}
	trueInterpreter := interpreter.copy()
	falseInterpreter := interpreter.copy()

	notCond := symbolic.NewUnaryOperation(condExpr, symbolic.UNARY_NOT)

	// Обновляем условия пути
	// Для true ветки: PathCondition AND cond
	trueInterpreter.PathCondition = symbolic.NewLogicalOperation(
		[]symbolic.SymbolicExpression{interpreter.PathCondition, condExpr},
		symbolic.AND,
	)

	// Для false ветки: PathCondition AND NOT(cond)
	falseInterpreter.PathCondition = symbolic.NewLogicalOperation(
		[]symbolic.SymbolicExpression{interpreter.PathCondition, notCond},
		symbolic.AND,
	)

	if len(instr.Block().Succs) >= 2 {
		trueInterpreter.currentBlock = instr.Block().Succs[0]
		trueInterpreter.instrIndex = 0

		falseInterpreter.currentBlock = instr.Block().Succs[1]
		falseInterpreter.instrIndex = 0
	} else {
		trueInterpreter.currentBlock = nil
		falseInterpreter.currentBlock = nil
	}

	return []Interpreter{trueInterpreter, falseInterpreter}
}

func (interpreter *Interpreter) interpretJump(instr *ssa.Jump) []Interpreter {
	if len(instr.Block().Succs) > 0 {
		interpreter.currentBlock = instr.Block().Succs[0]
		interpreter.instrIndex = 0
	} else {
		interpreter.currentBlock = nil
	}
	return []Interpreter{*interpreter}
}

func (interpreter *Interpreter) interpretUnOp(instr *ssa.UnOp) []Interpreter {
	operand := interpreter.resolveExpression(instr.X)

	// Определяем оператор
	var unaryOp symbolic.UnaryOperator
	switch instr.Op {
	case 1: // token.SUB (унарный минус)
		unaryOp = symbolic.UNARY_MINUS
	case 13: // token.NOT (логическое НЕ)
		unaryOp = symbolic.UNARY_NOT
	default:
		interpreter.instrIndex++
		return []Interpreter{*interpreter}
	}

	// Создаем унарную операцию
	result := symbolic.NewUnaryOperation(operand, unaryOp)

	// Сохраняем результат в локальную память
	frame := interpreter.getCurrentFrame()
	if frame != nil && instr.Name() != "" {
		frame.LocalMemory[instr.Name()] = result
	}

	interpreter.instrIndex++
	return []Interpreter{*interpreter}
}

func (interpreter *Interpreter) interpretBinOp(instr *ssa.BinOp) []Interpreter {
	// Получаем операнды
	left := interpreter.resolveExpression(instr.X)
	right := interpreter.resolveExpression(instr.Y)

	var binOp symbolic.BinaryOperator
	switch instr.Op {
	case token.ADD: // 1
		binOp = symbolic.ADD
	case token.SUB: // 2
		binOp = symbolic.SUB
	case token.MUL: // 3
		binOp = symbolic.MUL
	case token.QUO: // 4 (деление)
		binOp = symbolic.DIV
	case token.REM: // 5 (остаток)
		binOp = symbolic.MOD
	case token.EQL: // 6 (равно)
		binOp = symbolic.EQ
	case token.NEQ: // 7 (не равно)
		binOp = symbolic.NE
	case token.LSS: // 8 (меньше)
		binOp = symbolic.LT
	case token.LEQ: // 9 (меньше или равно)
		binOp = symbolic.LE
	case token.GTR: // 10 (больше)
		binOp = symbolic.GT
	case token.GEQ: // 11 (больше или равно)
		binOp = symbolic.GE
	default:
		fmt.Printf("Неизвестная бинарная операция: %v (числовое значение: %d)\n",
			instr.Op.String(), instr.Op)
		interpreter.instrIndex++
		return []Interpreter{*interpreter}
	}

	result := symbolic.NewBinaryOperation(left, right, binOp)

	//сохраняем результат в локальную память под именем инструкции
	frame := interpreter.getCurrentFrame()
	if frame != nil && instr.Name() != "" {
		frame.LocalMemory[instr.Name()] = result
	}

	interpreter.instrIndex++
	return []Interpreter{*interpreter}
}

func (interpreter *Interpreter) interpretAlloc(instr *ssa.Alloc) []Interpreter {
	ref := symbolic.NewRef(1, symbolic.RefType)

	frame := interpreter.getCurrentFrame()
	if frame != nil && instr.Name() != "" {
		frame.LocalMemory[instr.Name()] = ref
	}

	interpreter.instrIndex++
	return []Interpreter{*interpreter}
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
	}

	return symbolic.NewIntConstant(0)
}

func (interpreter *Interpreter) resolveUnOp(u *ssa.UnOp) symbolic.SymbolicExpression {
	// Сначала проверяем, есть ли значение в локальной памяти
	if u.Name() != "" {
		frame := interpreter.getCurrentFrame()
		if frame != nil {
			if expr, ok := frame.LocalMemory[u.Name()]; ok {
				return expr
			}
		}
	}

	// Если нет, вычисляем
	operand := interpreter.resolveExpression(u.X)

	var unaryOp symbolic.UnaryOperator
	switch u.Op {
	case 1: // token.SUB
		unaryOp = symbolic.UNARY_MINUS
	case 13: // token.NOT
		unaryOp = symbolic.UNARY_NOT
	default:
		return operand
	}

	return symbolic.NewUnaryOperation(operand, unaryOp)
}

func (interpreter *Interpreter) resolveBinOp(b *ssa.BinOp) symbolic.SymbolicExpression {
	fmt.Printf("=== ОТЛАДКА resolveBinOp ===\n")
	fmt.Printf("  Операция: %s, имя: %s, код операции: %d\n", b.String(), b.Name(), b.Op)

	if b.Name() != "" {
		frame := interpreter.getCurrentFrame()
		if frame != nil {
			if expr, ok := frame.LocalMemory[b.Name()]; ok {
				fmt.Printf("  Найдено в локальной памяти: %s\n", expr.String())
				return expr
			}
		}
	}

	left := interpreter.resolveExpression(b.X)
	right := interpreter.resolveExpression(b.Y)

	fmt.Printf("  Левый операнд: %s, правый операнд: %s\n", left.String(), right.String())

	var binOp symbolic.BinaryOperator
	switch b.Op {
	case 1: // token.ADD
		binOp = symbolic.ADD
	case 2: // token.SUB
		binOp = symbolic.SUB
	case 3: // token.MUL
		binOp = symbolic.MUL
	case 4: // token.QUO
		binOp = symbolic.DIV
	case 5: // token.REM
		binOp = symbolic.MOD
	case 6: // token.EQL
		binOp = symbolic.EQ
	case 7: // token.NEQ
		binOp = symbolic.NE
	case 8: // token.LSS
		binOp = symbolic.LT
	case 9: // token.LEQ
		binOp = symbolic.LE
	case 10: // token.GTR
		binOp = symbolic.GT
	case 11: // token.GEQ
		binOp = symbolic.GE
	default:
		fmt.Printf("  НЕИЗВЕСТНЫЙ ОПЕРАТОР: %d, left\n", b.Op)
		return left
	}

	fmt.Printf("  Создаем операцию: %s %s %s\n", left.String(), binOp.String(), right.String())

	result := symbolic.NewBinaryOperation(left, right, binOp)
	fmt.Printf("  Результат: %s (тип: %v)\n", result.String(), result.Type())

	if b.Name() != "" {
		frame := interpreter.getCurrentFrame()
		if frame != nil {
			fmt.Printf("  Сохраняем в локальную память как '%s'\n", b.Name())
			frame.LocalMemory[b.Name()] = result
		}
	}

	return result
}

func (interpreter *Interpreter) resolveParameter(p *ssa.Parameter) symbolic.SymbolicExpression {
	frame := interpreter.getCurrentFrame()
	if frame != nil {
		if val, ok := frame.LocalMemory[p.Name()]; ok {
			return val
		}
	}

	var exprType symbolic.ExpressionType
	switch p.Type().String() {
	case "int":
		exprType = symbolic.IntType
	case "bool":
		exprType = symbolic.BoolType
	default:
		exprType = symbolic.IntType
	}

	return symbolic.NewSymbolicVariable(p.Name(), exprType)
}

func (interpreter *Interpreter) resolveAlloc(a *ssa.Alloc) symbolic.SymbolicExpression {
	frame := interpreter.getCurrentFrame()
	if frame != nil && a.Name() != "" {
		if val, ok := frame.LocalMemory[a.Name()]; ok {
			return val
		}
	}

	return symbolic.NewRef(0, symbolic.RefType)
}

func (interpreter *Interpreter) resolvePhi(phi *ssa.Phi) symbolic.SymbolicExpression {
	frame := interpreter.getCurrentFrame()
	if frame != nil && phi.Name() != "" {
		if val, ok := frame.LocalMemory[phi.Name()]; ok {
			return val
		}
	}

	// Используем первое значение из ребер
	if len(phi.Edges) > 0 {
		return interpreter.resolveExpression(phi.Edges[0])
	}

	return symbolic.NewIntConstant(0)
}

// String возвращает строковое представление интерпретатора
func (interpreter *Interpreter) String() string {
	result := fmt.Sprintf("Interpreter:\n")
	result += fmt.Sprintf("  PathCondition: %s\n", interpreter.PathCondition.String())
	result += fmt.Sprintf("  CurrentBlock: %v\n", interpreter.currentBlock)
	result += fmt.Sprintf("  InstrIndex: %d\n", interpreter.instrIndex)

	if len(interpreter.CallStack) > 0 {
		frame := interpreter.getCurrentFrame()
		if frame != nil && frame.ReturnValue != nil {
			result += fmt.Sprintf("  ReturnValue: %s\n", frame.ReturnValue.String())
		}
	}

	return result
}
