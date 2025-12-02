package symbolic

import "fmt"

type DebugVisitor struct {
	Indent int
}

func (dv *DebugVisitor) VisitVariable(expr *SymbolicVariable) interface{} {
	dv.printIndent("Variable: " + expr.Name + " (" + expr.Type().String() + ")")
	return nil
}

func (dv *DebugVisitor) VisitIntConstant(expr *IntConstant) interface{} {
	dv.printIndent("IntConstant: " + expr.String())
	return nil
}

func (dv *DebugVisitor) VisitBoolConstant(expr *BoolConstant) interface{} {
	dv.printIndent("BoolConstant: " + expr.String())
	return nil
}

func (dv *DebugVisitor) VisitBinaryOperation(expr *BinaryOperation) interface{} {
	dv.printIndent("BinaryOperation: " + expr.Operator.String())
	dv.Indent++
	expr.Left.Accept(dv)
	expr.Right.Accept(dv)
	dv.Indent--
	return nil
}

func (dv *DebugVisitor) VisitLogicalOperation(expr *LogicalOperation) interface{} {
	dv.printIndent("LogicalOperation: " + expr.Operator.String())
	dv.Indent++
	for i, op := range expr.Operands {
		dv.printIndent(fmt.Sprintf("Operand[%d]:", i))
		op.Accept(dv)
	}
	dv.Indent--
	return nil
}

func (dv *DebugVisitor) VisitUnaryOperation(expr *UnaryOperation) interface{} {
	dv.printIndent("UnaryOperation: " + expr.Operator.String())
	dv.Indent++
	expr.Operand.Accept(dv)
	dv.Indent--
	return nil
}

func (dv *DebugVisitor) VisitRef(expr *Ref) interface{} {
	dv.printIndent("Ref: " + expr.String() + " (" + expr.Type().String() + ")")
	return nil
}

func (dv *DebugVisitor) VisitFieldAddr(expr *FieldAddr) interface{} {
	dv.printIndent("FieldAddr: " + expr.String())
	dv.Indent++
	expr.Ref.Accept(dv)
	dv.printIndent(fmt.Sprintf("FieldIndex: %d", expr.FieldIndex))
	dv.Indent--
	return nil
}

func (dv *DebugVisitor) VisitIndexAddr(expr *IndexAddr) interface{} {
	dv.printIndent("IndexAddr: " + expr.String())
	dv.Indent++
	expr.Ref.Accept(dv)
	dv.printIndent(fmt.Sprintf("Index: %d", expr.Index))
	dv.Indent--
	return nil
}

func (dv *DebugVisitor) printIndent(msg string) {
	for i := 0; i < dv.Indent; i++ {
		fmt.Print("  ")
	}
	fmt.Println(msg)
}

func NewDebugVisitor() *DebugVisitor {
	return &DebugVisitor{Indent: 0}
}
