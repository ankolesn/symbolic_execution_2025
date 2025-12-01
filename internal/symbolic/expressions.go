// Package symbolic содержит конкретные реализации символьных выражений
package symbolic

import "fmt"

// Операторы для бинарных выражений
type BinaryOperator int

const (
	// Арифметические операторы
	ADD BinaryOperator = iota
	SUB
	MUL
	DIV
	MOD

	// Операторы сравнения
	EQ // равно
	NE // не равно
	LT // меньше
	LE // меньше или равно
	GT // больше
	GE // больше или равно
)

func (op BinaryOperator) String() string {
	switch op {
	case ADD:
		return "+"
	case SUB:
		return "-"
	case MUL:
		return "*"
	case DIV:
		return "/"
	case MOD:
		return "%"
	case EQ:
		return "=="
	case NE:
		return "!="
	case LT:
		return "<"
	case LE:
		return "<="
	case GT:
		return ">"
	case GE:
		return ">="
	default:
		return "unknown"
	}
}

type LogicalOperator int

const (
	AND LogicalOperator = iota
	OR
	NOT
	IMPLIES
)

func (op LogicalOperator) String() string {
	switch op {
	case AND:
		return "&&"
	case OR:
		return "||"
	case NOT:
		return "!"
	case IMPLIES:
		return "=>"
	default:
		return "unknown"
	}
}

type UnaryOperator int

const (
	UNARY_MINUS UnaryOperator = iota // -x
	UNARY_NOT                        // !x или not x
)

func (op UnaryOperator) String() string {
	switch op {
	case UNARY_MINUS:
		return "-"
	case UNARY_NOT:
		return "!"
	default:
		return "unknown"
	}
}

// SymbolicExpression - базовый интерфейс для всех символьных выражений
type SymbolicExpression interface {
	// Type возвращает тип выражения
	Type() ExpressionType

	// String возвращает строковое представление выражения
	String() string

	// Accept принимает visitor для обхода дерева выражений
	Accept(visitor Visitor) interface{}
}

// SymbolicVariable представляет символьную переменную
type SymbolicVariable struct {
	Name     string
	ExprType ExpressionType
}

// NewSymbolicVariable создаёт новую символьную переменную
func NewSymbolicVariable(name string, exprType ExpressionType) *SymbolicVariable {
	return &SymbolicVariable{
		Name:     name,
		ExprType: exprType,
	}
}

// Type возвращает тип переменной
func (sv *SymbolicVariable) Type() ExpressionType {
	return sv.ExprType
}

// String возвращает строковое представление переменной
func (sv *SymbolicVariable) String() string {
	return sv.Name
}

// Accept реализует Visitor pattern
func (sv *SymbolicVariable) Accept(visitor Visitor) interface{} {
	return visitor.VisitVariable(sv)
}

// IntConstant представляет целочисленную константу
type IntConstant struct {
	Value int64
}

// NewIntConstant создаёт новую целочисленную константу
func NewIntConstant(value int64) *IntConstant {
	return &IntConstant{Value: value}
}

// Type возвращает тип константы
func (ic *IntConstant) Type() ExpressionType {
	return IntType
}

// String возвращает строковое представление константы
func (ic *IntConstant) String() string {
	return fmt.Sprintf("%d", ic.Value)
}

// Accept реализует Visitor pattern
func (ic *IntConstant) Accept(visitor Visitor) interface{} {
	return visitor.VisitIntConstant(ic)
}

// BoolConstant представляет булеву константу
type BoolConstant struct {
	Value bool
}

// NewBoolConstant создаёт новую булеву константу
func NewBoolConstant(value bool) *BoolConstant {
	return &BoolConstant{Value: value}
}

// Type возвращает тип константы
func (bc *BoolConstant) Type() ExpressionType {
	return BoolType
}

// String возвращает строковое представление константы
func (bc *BoolConstant) String() string {
	return fmt.Sprintf("%t", bc.Value)
}

// Accept реализует Visitor pattern
func (bc *BoolConstant) Accept(visitor Visitor) interface{} {
	return visitor.VisitBoolConstant(bc)
}

// BinaryOperation представляет бинарную операцию
type BinaryOperation struct {
	Left     SymbolicExpression
	Right    SymbolicExpression
	Operator BinaryOperator
}

// NewBinaryOperation создаёт новую бинарную операцию
func NewBinaryOperation(left, right SymbolicExpression, op BinaryOperator) *BinaryOperation {
	switch op {
	case ADD, SUB, MUL, DIV, MOD:
		if left.Type() != IntType || right.Type() != IntType {
			panic("Арифметические операции требуют целочисленные операнды")
		}
	case EQ, NE:
		if left.Type() != right.Type() {
			panic("Операторы сравнения требуют операнды одного типа")
		}
	case LT, LE, GT, GE:
		if left.Type() != IntType || right.Type() != IntType {
			panic("Операторы сравнения требуют целочисленные операнды")
		}
	}

	return &BinaryOperation{
		Left:     left,
		Right:    right,
		Operator: op,
	}
}

// Type возвращает результирующий тип операции
func (bo *BinaryOperation) Type() ExpressionType {
	switch bo.Operator {
	case ADD, SUB, MUL, DIV, MOD:
		return IntType
	case EQ, NE, LT, LE, GT, GE:
		return BoolType
	default:
		panic("Неизвестный оператор")
	}
}

// String возвращает строковое представление операции
func (bo *BinaryOperation) String() string {
	return fmt.Sprintf("(%s %s %s)", bo.Left.String(), bo.Operator.String(), bo.Right.String())
}

// Accept реализует Visitor pattern
func (bo *BinaryOperation) Accept(visitor Visitor) interface{} {
	return visitor.VisitBinaryOperation(bo)
}

// LogicalOperation представляет логическую операцию
type LogicalOperation struct {
	Operands []SymbolicExpression
	Operator LogicalOperator
}

// NewLogicalOperation создаёт новую логическую операцию
func NewLogicalOperation(operands []SymbolicExpression, op LogicalOperator) *LogicalOperation {
	// Проверка количества операндов
	if op == NOT && len(operands) != 1 {
		panic("Оператор NOT требует один операнд")
	}
	if (op == AND || op == OR || op == IMPLIES) && len(operands) < 2 {
		panic("Логические операторы AND, OR, IMPLIES требуют как минимум два операнда")
	}

	// Проверка типов операндов
	for i, operand := range operands {
		if operand == nil {
			panic(fmt.Sprintf("Операнд %d равен nil", i))
		}
		if operand.Type() != BoolType {
			fmt.Printf("Отладка NewLogicalOperation: операнд %d имеет тип %v, значение: %s\n",
				i, operand.Type(), operand.String())
			panic("Логические операции требуют булевы операнды")
		}
	}

	return &LogicalOperation{
		Operands: operands,
		Operator: op,
	}
}

// Type возвращает тип логической операции (всегда bool)
func (lo *LogicalOperation) Type() ExpressionType {
	return BoolType
}

// String возвращает строковое представление логической операции
func (lo *LogicalOperation) String() string {
	switch lo.Operator {
	case NOT:
		return fmt.Sprintf("!%s", lo.Operands[0].String())
	case AND, OR:
		operatorStr := lo.Operator.String()
		result := "("
		for i, operand := range lo.Operands {
			if i > 0 {
				result += " " + operatorStr + " "
			}
			result += operand.String()
		}
		result += ")"
		return result
	case IMPLIES:
		if len(lo.Operands) != 2 {
			panic("IMPLIES требует два операнда")
		}
		return fmt.Sprintf("(%s => %s)", lo.Operands[0].String(), lo.Operands[1].String())
	default:
		panic("Неизвестный логический оператор")
	}
}

// Accept реализует Visitor pattern
func (lo *LogicalOperation) Accept(visitor Visitor) interface{} {
	return visitor.VisitLogicalOperation(lo)
}

// UnaryOperation представляет унарную операцию
type UnaryOperation struct {
	Operand  SymbolicExpression
	Operator UnaryOperator
}

// NewUnaryOperation создаёт новую унарную операцию
func NewUnaryOperation(operand SymbolicExpression, op UnaryOperator) *UnaryOperation {
	switch op {
	case UNARY_MINUS:
		if operand.Type() != IntType {
			panic("Унарный минус требует целочисленный операнд")
		}
	case UNARY_NOT:
		if operand.Type() != BoolType {
			panic("Логическое НЕ требует булев операнд")
		}
	}

	return &UnaryOperation{
		Operand:  operand,
		Operator: op,
	}
}

// Type возвращает тип операции
func (uo *UnaryOperation) Type() ExpressionType {
	return uo.Operand.Type()
}

// String возвращает строковое представление операции
func (uo *UnaryOperation) String() string {
	return fmt.Sprintf("%s%s", uo.Operator.String(), uo.Operand.String())
}

// Accept реализует Visitor pattern
func (uo *UnaryOperation) Accept(visitor Visitor) interface{} {
	return visitor.VisitUnaryOperation(uo)
}

type Ref struct {
	ID       int
	ExprType ExpressionType
}

func NewRef(id int, exprType ExpressionType) *Ref {
	return &Ref{
		ID:       id,
		ExprType: exprType,
	}
}

func (r *Ref) Type() ExpressionType {
	return RefType
}

func (r *Ref) String() string {
	return fmt.Sprintf("ref_%d", r.ID)
}

func (r *Ref) Accept(visitor Visitor) interface{} {
	return visitor.VisitRef(r)
}
