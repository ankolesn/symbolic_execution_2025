// Package translator содержит реализацию транслятора в Z3
package translator

import (
	"fmt"

	"symbolic-execution-course/internal/symbolic"

	"github.com/ebukreev/go-z3/z3"
)

// Z3Translator транслирует символьные выражения в Z3 формулы
type Z3Translator struct {
	ctx    *z3.Context
	config *z3.Config
	vars   map[string]z3.Value // Кэш переменных
}

// NewZ3Translator создаёт новый экземпляр Z3 транслятора
func NewZ3Translator() *Z3Translator {
	config := &z3.Config{}
	ctx := z3.NewContext(config)

	return &Z3Translator{
		ctx:    ctx,
		config: config,
		vars:   make(map[string]z3.Value),
	}
}

// GetContext возвращает Z3 контекст
func (zt *Z3Translator) GetContext() interface{} {
	return zt.ctx
}

// Reset сбрасывает состояние транслятора
func (zt *Z3Translator) Reset() {
	zt.vars = make(map[string]z3.Value)
}

// Close освобождает ресурсы
func (zt *Z3Translator) Close() {
	// Z3 контекст закрывается автоматически
}

// TranslateExpression транслирует символьное выражение в Z3
func (zt *Z3Translator) TranslateExpression(expr symbolic.SymbolicExpression) (interface{}, error) {
	result := expr.Accept(zt)
	if result == nil {
		return nil, fmt.Errorf("трансляция вернула nil")
	}
	return result, nil
}

// VisitVariable транслирует символьную переменную в Z3
func (zt *Z3Translator) VisitVariable(expr *symbolic.SymbolicVariable) interface{} {
	// Проверить, есть ли переменная в кэше
	if v, exists := zt.vars[expr.Name]; exists {
		return v
	}

	// Создать новую Z3 переменную соответствующего типа
	var z3Var z3.Value
	switch expr.Type() {
	case symbolic.IntType:
		z3Var = zt.ctx.IntConst(expr.Name)
	case symbolic.BoolType:
		z3Var = zt.ctx.BoolConst(expr.Name)
	default:
		fmt.Printf("Warning: неподдерживаемый тип переменной: %v\n", expr.Type())
		return nil
	}

	// Добавить в кэш и вернуть
	zt.vars[expr.Name] = z3Var
	return z3Var
}

// VisitIntConstant транслирует целочисленную константу в Z3
func (zt *Z3Translator) VisitIntConstant(expr *symbolic.IntConstant) interface{} {
	// Создать Z3 константу с помощью zt.ctx.FromBigInt или аналогичного метода
	return zt.ctx.FromInt(int64(expr.Value), zt.ctx.IntSort())
}

// VisitBoolConstant транслирует булеву константу в Z3
func (zt *Z3Translator) VisitBoolConstant(expr *symbolic.BoolConstant) interface{} {
	// Использовать zt.ctx.FromBool для создания Z3 булевой константы
	return zt.ctx.FromBool(expr.Value)
}

// VisitRef транслирует символьную ссылку в Z3
func (zt *Z3Translator) VisitRef(expr *symbolic.Ref) interface{} {
	// Представляем ссылку как целочисленную константу с ID ссылки
	return zt.ctx.FromInt(int64(expr.ID), zt.ctx.IntSort())
}

// VisitBinaryOperation транслирует бинарную операцию в Z3
func (zt *Z3Translator) VisitBinaryOperation(expr *symbolic.BinaryOperation) interface{} {
	// Транслировать левый и правый операнды
	left := expr.Left.Accept(zt)
	right := expr.Right.Accept(zt)

	if left == nil || right == nil {
		return nil
	}

	// В зависимости от оператора создать соответствующую Z3 операцию
	switch expr.Operator {
	case symbolic.ADD:
		return left.(z3.Int).Add(right.(z3.Int))
	case symbolic.SUB:
		return left.(z3.Int).Sub(right.(z3.Int))
	case symbolic.MUL:
		return left.(z3.Int).Mul(right.(z3.Int))
	case symbolic.DIV:
		return left.(z3.Int).Div(right.(z3.Int))
	case symbolic.MOD:
		return left.(z3.Int).Mod(right.(z3.Int))
	case symbolic.EQ:
		// Для равенства используем метод Eq
		if expr.Left.Type() == symbolic.BoolType {
			return left.(z3.Bool).Eq(right.(z3.Bool))
		} else {
			return left.(z3.Int).Eq(right.(z3.Int))
		}
	case symbolic.NE:
		// Для неравенства используем Not от равенства
		if expr.Left.Type() == symbolic.BoolType {
			return left.(z3.Bool).Eq(right.(z3.Bool)).Not()
		} else {
			return left.(z3.Int).Eq(right.(z3.Int)).Not()
		}
	case symbolic.LT:
		return left.(z3.Int).LT(right.(z3.Int))
	case symbolic.LE:
		return left.(z3.Int).LE(right.(z3.Int))
	case symbolic.GT:
		return left.(z3.Int).GT(right.(z3.Int))
	case symbolic.GE:
		return left.(z3.Int).GE(right.(z3.Int))
	default:
		fmt.Printf("Warning: неизвестный бинарный оператор: %v\n", expr.Operator)
		return nil
	}
}

// VisitLogicalOperation транслирует логическую операцию в Z3
func (zt *Z3Translator) VisitLogicalOperation(expr *symbolic.LogicalOperation) interface{} {
	// 1. Транслировать все операнды
	operands := make([]z3.Bool, len(expr.Operands))
	for i, op := range expr.Operands {
		result := op.Accept(zt)
		operands[i] = result.(z3.Bool)
	}

	switch expr.Operator {
	case symbolic.AND:
		result := operands[0]
		for i := 1; i < len(operands); i++ {
			result = result.And(operands[i])
		}
		return result
	case symbolic.OR:
		result := operands[0]
		for i := 1; i < len(operands); i++ {
			result = result.Or(operands[i])
		}
		return result
	case symbolic.NOT:
		if len(operands) != 1 {
			fmt.Printf("Error: NOT требует ровно один операнд\n")
			return nil
		}
		return operands[0].Not()
	case symbolic.IMPLIES:
		if len(operands) != 2 {
			fmt.Printf("Error: IMPLIES требует два операнд\n")
			return nil
		}
		return operands[0].Implies(operands[1])
	default:
		fmt.Printf("Warning: неизвестный логический оператор: %v\n", expr.Operator)
		return nil
	}
}

func (zt *Z3Translator) VisitUnaryOperation(expr *symbolic.UnaryOperation) interface{} {
	operand := expr.Operand.Accept(zt)
	if operand == nil {
		return nil
	}

	switch expr.Operator {
	case symbolic.UNARY_MINUS:
		return operand.(z3.Int).Neg()
	case symbolic.UNARY_NOT:
		return operand.(z3.Bool).Not()
	default:
		fmt.Printf("Warning: неизвестный унарный оператор: %v\n", expr.Operator)
		return nil
	}
}
