// Демонстрационная программа для тестирования символьных выражений
package main

import (
	"fmt"
	"log"

	"symbolic-execution-course/internal/symbolic"
	"symbolic-execution-course/internal/translator"
)

func main() {
	fmt.Println("=== Symbolic Expressions Demo ===")

		// Создаём простые символьные выражения
		x := symbolic.NewSymbolicVariable("x", symbolic.IntType)
		y := symbolic.NewSymbolicVariable("y", symbolic.IntType)
		five := symbolic.NewIntConstant(5)

		// Создаём выражение: x + y > 5
		sum := symbolic.NewBinaryOperation(x, y, symbolic.ADD)
		condition := symbolic.NewBinaryOperation(sum, five, symbolic.GT)

		fmt.Printf("Выражение: %s\n", condition.String())
		fmt.Printf("Тип выражения: %s\n", condition.Type().String())

		// Демонстрация структуры выражения с помощью DebugVisitor
		debugVisitor := &symbolic.DebugVisitor{}
		fmt.Println("Структура выражения:")
		condition.Accept(debugVisitor)

		// Создаём Z3 транслятор
		translator := translator.NewZ3Translator()
		defer translator.Close()

		// Транслируем в Z3
		z3Expr, err := translator.TranslateExpression(condition)
		if err != nil {
			log.Fatalf("Ошибка трансляции: %v", err)
		}

		fmt.Printf("Z3 выражение создано: %T\n", z3Expr)

		// Создаём более сложное выражение: (x > 0) && (y < 10)
		zero := symbolic.NewIntConstant(0)
		ten := symbolic.NewIntConstant(10)

		cond1 := symbolic.NewBinaryOperation(x, zero, symbolic.GT)
		cond2 := symbolic.NewBinaryOperation(y, ten, symbolic.LT)

		andExpr := symbolic.NewLogicalOperation([]symbolic.SymbolicExpression{cond1, cond2}, symbolic.AND)

		fmt.Printf("Сложное выражение: %s\n", andExpr.String())

		fmt.Println("Структура сложного выражения:")
		andExpr.Accept(debugVisitor)

		// Транслируем сложное выражение
		z3AndExpr, err := translator.TranslateExpression(andExpr)
		if err != nil {
			log.Fatalf("Ошибка трансляции сложного выражения: %v", err)
		}

	fmt.Printf("Сложное Z3 выражение создано: %T\n", z3AndExpr)

	fmt.Println("\n=== Тестирование унарных операций ===")

	negX := symbolic.NewUnaryOperation(x, symbolic.UNARY_MINUS)
	fmt.Printf("Унарный минус: %s\n", negX.String())

	b := symbolic.NewSymbolicVariable("b", symbolic.BoolType)
	notB := symbolic.NewUnaryOperation(b, symbolic.UNARY_NOT)
	fmt.Printf("Логическое НЕ: %s\n", notB.String())

	z3NegX, _ := translator.TranslateExpression(negX)
	fmt.Printf("Z3 унарный минус создан: %T\n", z3NegX)

	z3NotB, _ := translator.TranslateExpression(notB)
	fmt.Printf("Z3 логическое НЕ создано: %T\n", z3NotB)

}