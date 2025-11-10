package memory

import (
	"symbolic-execution-course/internal/symbolic"
	"symbolic-execution-course/internal/translator"
	"testing"

	"github.com/ebukreev/go-z3/z3"
)

// TestStructBasic тестирует создание и инициализацию структуры
func TestStructBasic(t *testing.T) {
	mem := NewSymbolicMemory()

	person := mem.Allocate(symbolic.StructType)

	name := symbolic.NewSymbolicVariable("name", symbolic.IntType)
	age := symbolic.NewIntConstant(25)
	id := symbolic.NewIntConstant(1001)

	mem.AssignField(person, 0, name)
	mem.AssignField(person, 1, age)
	mem.AssignField(person, 2, id)

	if mem.GetFieldValue(person, 0).String() != "name" {
		t.Errorf("Expected name, got %s", mem.GetFieldValue(person, 0).String())
	}
	if mem.GetFieldValue(person, 1).String() != "25" {
		t.Errorf("Expected 25, got %s", mem.GetFieldValue(person, 1).String())
	}
	if mem.GetFieldValue(person, 2).String() != "1001" {
		t.Errorf("Expected 1001, got %s", mem.GetFieldValue(person, 2).String())
	}
}

// TestStructModification тестирует модификацию структуры
func TestStructModification(t *testing.T) {
	mem := NewSymbolicMemory()

	person := mem.Allocate(symbolic.StructType)
	initialAge := symbolic.NewIntConstant(25)
	initialID := symbolic.NewIntConstant(1001)

	mem.AssignField(person, 1, initialAge)
	mem.AssignField(person, 2, initialID)

	newAge := symbolic.NewBinaryOperation(
		mem.GetFieldValue(person, 1),
		symbolic.NewIntConstant(1),
		symbolic.ADD,
	)
	newID := symbolic.NewBinaryOperation(
		mem.GetFieldValue(person, 2),
		symbolic.NewIntConstant(2),
		symbolic.MUL,
	)

	mem.AssignField(person, 1, newAge)
	mem.AssignField(person, 2, newID)

	if mem.GetFieldValue(person, 1).String() != "(25 + 1)" {
		t.Errorf("Expected (25 + 1), got %s", mem.GetFieldValue(person, 1).String())
	}
	if mem.GetFieldValue(person, 2).String() != "(1001 * 2)" {
		t.Errorf("Expected (1001 * 2), got %s", mem.GetFieldValue(person, 2).String())
	}
}

// TestArrayFixed тестирует работу с фиксированным массивом
func TestArrayFixed(t *testing.T) {
	mem := NewSymbolicMemory()

	arr := mem.Allocate(symbolic.ArrayType)

	for i := 0; i < 5; i++ {
		value := symbolic.NewBinaryOperation(
			symbolic.NewIntConstant(int64(i)),
			symbolic.NewIntConstant(int64(i)),
			symbolic.MUL,
		)
		mem.AssignToArray(arr, i, value)
	}

	for i := 0; i < 5; i++ {
		expected := symbolic.NewBinaryOperation(
			symbolic.NewIntConstant(int64(i)),
			symbolic.NewIntConstant(int64(i)),
			symbolic.MUL,
		).String()

		if mem.GetFromArray(arr, i).String() != expected {
			t.Errorf("arr[%d]: expected %s, got %s", i, expected, mem.GetFromArray(arr, i).String())
		}
	}
}

// TestArrayModification тестирует модификацию массива
func TestArrayModification(t *testing.T) {
	mem := NewSymbolicMemory()

	arr := mem.Allocate(symbolic.ArrayType)
	for i := 0; i < 5; i++ {
		mem.AssignToArray(arr, i, symbolic.NewIntConstant(int64(i*2)))
	}

	for i := 0; i < 5; i++ {
		oldValue := mem.GetFromArray(arr, i)
		newValue := symbolic.NewBinaryOperation(oldValue, symbolic.NewIntConstant(1), symbolic.ADD)
		mem.AssignToArray(arr, i, newValue)
	}

	for i := 0; i < 5; i++ {
		expected := symbolic.NewBinaryOperation(
			symbolic.NewIntConstant(int64(i*2)),
			symbolic.NewIntConstant(1),
			symbolic.ADD,
		).String()

		if mem.GetFromArray(arr, i).String() != expected {
			t.Errorf("arr[%d]: expected %s, got %s", i, expected, mem.GetFromArray(arr, i).String())
		}
	}
}

// TestStructWithArray тестирует структуру с массивом внутри
func TestStructWithArray(t *testing.T) {
	mem := NewSymbolicMemory()

	student := mem.Allocate(symbolic.StructType)

	mem.AssignField(student, 0, symbolic.NewSymbolicVariable("charlie", symbolic.IntType))

	grades := mem.Allocate(symbolic.ArrayType)
	gradeValues := []int64{85, 90, 78, 92, 88}
	for i, grade := range gradeValues {
		mem.AssignToArray(grades, i, symbolic.NewIntConstant(grade))
	}

	mem.AssignField(student, 1, symbolic.NewRef(grades.ID, symbolic.ArrayType))

	var sum symbolic.SymbolicExpression = symbolic.NewIntConstant(0)
	for i := 0; i < 5; i++ {
		grade := mem.GetFromArray(grades, i)
		sum = symbolic.NewBinaryOperation(sum, grade, symbolic.ADD)
	}

	average := symbolic.NewBinaryOperation(
		sum,
		symbolic.NewIntConstant(5),
		symbolic.DIV,
	)
	mem.AssignField(student, 2, average)

	// Проверяем структуру
	if mem.GetFieldValue(student, 0).String() != "charlie" {
		t.Errorf("Expected charlie, got %s", mem.GetFieldValue(student, 0).String())
	}

	gradesRef := mem.GetFieldValue(student, 1).(*symbolic.Ref)
	if mem.GetFromArray(gradesRef, 0).String() != "85" {
		t.Errorf("Expected 85, got %s", mem.GetFromArray(gradesRef, 0).String())
	}
}

// TestNestedStructs тестирует вложенные структуры
func TestNestedStructs(t *testing.T) {
	mem := NewSymbolicMemory()

	employee := mem.Allocate(symbolic.StructType)

	person := mem.Allocate(symbolic.StructType)
	mem.AssignField(person, 0, symbolic.NewSymbolicVariable("david", symbolic.IntType))
	mem.AssignField(person, 1, symbolic.NewIntConstant(35))
	mem.AssignField(person, 2, symbolic.NewIntConstant(3003))

	address := mem.Allocate(symbolic.StructType)
	mem.AssignField(address, 0, symbolic.NewSymbolicVariable("main_st", symbolic.IntType))
	mem.AssignField(address, 1, symbolic.NewSymbolicVariable("boston", symbolic.IntType))
	mem.AssignField(address, 2, symbolic.NewIntConstant(12345))

	mem.AssignField(employee, 0, symbolic.NewRef(person.ID, symbolic.StructType))
	mem.AssignField(employee, 1, symbolic.NewRef(address.ID, symbolic.StructType))
	mem.AssignField(employee, 2, symbolic.NewIntConstant(75000))

	personRef := mem.GetFieldValue(employee, 0).(*symbolic.Ref)
	if mem.GetFieldValue(personRef, 0).String() != "david" {
		t.Errorf("Expected david, got %s", mem.GetFieldValue(personRef, 0).String())
	}

	addressRef := mem.GetFieldValue(employee, 1).(*symbolic.Ref)
	if mem.GetFieldValue(addressRef, 2).String() != "12345" {
		t.Errorf("Expected 12345, got %s", mem.GetFieldValue(addressRef, 2).String())
	}

	if mem.GetFieldValue(employee, 2).String() != "75000" {
		t.Errorf("Expected 75000, got %s", mem.GetFieldValue(employee, 2).String())
	}
}

// TestArrayOfStructs тестирует массив структур
func TestArrayOfStructs(t *testing.T) {
	mem := NewSymbolicMemory()

	people := mem.Allocate(symbolic.ArrayType)

	for i := 0; i < 3; i++ {
		person := mem.Allocate(symbolic.StructType)
		mem.AssignField(person, 0, symbolic.NewSymbolicVariable("name", symbolic.IntType))
		mem.AssignField(person, 1, symbolic.NewIntConstant(int64(25+i*5)))
		mem.AssignField(person, 2, symbolic.NewIntConstant(int64(i+1)))

		mem.AssignToArray(people, i, symbolic.NewRef(person.ID, symbolic.StructType))
	}

	secondPersonRef := mem.GetFromArray(people, 1).(*symbolic.Ref)
	oldAge := mem.GetFieldValue(secondPersonRef, 1)
	newAge := symbolic.NewBinaryOperation(oldAge, symbolic.NewIntConstant(5), symbolic.ADD)
	mem.AssignField(secondPersonRef, 1, newAge)

	if mem.GetFieldValue(secondPersonRef, 1).String() != "(30 + 5)" {
		t.Errorf("Expected (30 + 5), got %s", mem.GetFieldValue(secondPersonRef, 1).String())
	}
}

// TestPathConstraintMutability тестирует условную логику с мутабельностью
func TestPathConstraintMutability(t *testing.T) {
	mem := NewSymbolicMemory()

	person := mem.Allocate(symbolic.StructType)
	ageVar := symbolic.NewSymbolicVariable("age", symbolic.IntType)
	mem.AssignField(person, 1, ageVar)

	currentAge := mem.GetFieldValue(person, 1)
	condition := symbolic.NewBinaryOperation(currentAge, symbolic.NewIntConstant(18), symbolic.NE)

	// В символьном выполнении мы рассматриваем оба пути
	// Для теста предположим, что условие истинно
	if condition.String() == "(age != 18)" {
		mem.AssignField(person, 1, symbolic.NewIntConstant(18))

		newAge := mem.GetFieldValue(person, 1)
		if newAge.String() != "18" {
			t.Errorf("After setting age to 18, expected 18, got %s", newAge.String())
		}
	}
}

// TestBasicAliasing тестирует базовый сценарий алиасинга
func TestBasicAliasing(t *testing.T) {
	mem := NewSymbolicMemory()

	foo1 := mem.Allocate(symbolic.StructType)
	foo2 := mem.Allocate(symbolic.StructType)

	mem.AssignField(foo1, 0, symbolic.NewIntConstant(0))
	mem.AssignField(foo2, 0, symbolic.NewIntConstant(0))

	mem.AssignField(foo2, 0, symbolic.NewIntConstant(5))

	mem.AssignField(foo1, 0, symbolic.NewIntConstant(2))

	if mem.GetFieldValue(foo1, 0).String() != "2" {
		t.Errorf("foo1.a should be 2, got %s", mem.GetFieldValue(foo1, 0).String())
	}
	if mem.GetFieldValue(foo2, 0).String() != "5" {
		t.Errorf("foo2.a should be 5, got %s", mem.GetFieldValue(foo2, 0).String())
	}
}

// TestAliasingWithCreateAlias тестирует алиасинг с использованием CreateAlias
func TestAliasingWithCreateAlias(t *testing.T) {
	mem := NewSymbolicMemory()

	original := mem.Allocate(symbolic.StructType)
	mem.AssignField(original, 0, symbolic.NewIntConstant(10))

	alias := mem.CreateAlias(original, 100)

	mem.AssignField(alias, 0, symbolic.NewIntConstant(20))

	if mem.GetFieldValue(original, 0).String() != "20" {
		t.Errorf("Expected 20, got %s", mem.GetFieldValue(original, 0).String())
	}
}

func TestAllocateStruct(t *testing.T) {
	mem := NewSymbolicMemory()

	person := mem.AllocateStruct(3)

	if person.ExprType != symbolic.StructType {
		t.Errorf("Expected StructType, got %v", person.ExprType)
	}

	for i := 0; i < 3; i++ {
		fieldValue := mem.GetFieldValue(person, i)
		if fieldValue.String() != "0" {
			t.Errorf("Field %d: expected 0, got %s", i, fieldValue.String())
		}
	}

	mem.AssignField(person, 0, symbolic.NewSymbolicVariable("name", symbolic.IntType))
	mem.AssignField(person, 1, symbolic.NewIntConstant(25))
	mem.AssignField(person, 2, symbolic.NewIntConstant(1001))

	if mem.GetFieldValue(person, 0).String() != "name" {
		t.Errorf("Field 0: expected 'name', got %s", mem.GetFieldValue(person, 0).String())
	}
	if mem.GetFieldValue(person, 1).String() != "25" {
		t.Errorf("Field 1: expected '25', got %s", mem.GetFieldValue(person, 1).String())
	}
	if mem.GetFieldValue(person, 2).String() != "1001" {
		t.Errorf("Field 2: expected '1001', got %s", mem.GetFieldValue(person, 2).String())
	}
}

func TestAllocateArray(t *testing.T) {
	mem := NewSymbolicMemory()

	arr := mem.AllocateArray(5)

	if arr.ExprType != symbolic.ArrayType {
		t.Errorf("Expected ArrayType, got %v", arr.ExprType)
	}

	for i := 0; i < 5; i++ {
		elemValue := mem.GetFromArray(arr, i)
		if elemValue.String() != "0" {
			t.Errorf("Element %d: expected 0, got %s", i, elemValue.String())
		}
	}

	for i := 0; i < 5; i++ {
		value := symbolic.NewBinaryOperation(
			symbolic.NewIntConstant(int64(i)),
			symbolic.NewIntConstant(int64(i)),
			symbolic.MUL,
		)
		mem.AssignToArray(arr, i, value)
	}

	for i := 0; i < 5; i++ {
		expected := symbolic.NewBinaryOperation(
			symbolic.NewIntConstant(int64(i)),
			symbolic.NewIntConstant(int64(i)),
			symbolic.MUL,
		).String()

		actual := mem.GetFromArray(arr, i).String()
		if actual != expected {
			t.Errorf("arr[%d]: expected %s, got %s", i, expected, actual)
		}
	}
}

func TestEdgeCases(t *testing.T) {
	mem := NewSymbolicMemory()

	// Тест с нулевым количеством полей
	emptyStruct := mem.AllocateStruct(0)
	if emptyStruct.ExprType != symbolic.StructType {
		t.Errorf("Empty struct should have StructType")
	}

	// Тест с нулевой длиной массива
	emptyArray := mem.AllocateArray(0)
	if emptyArray.ExprType != symbolic.ArrayType {
		t.Errorf("Empty array should have ArrayType")
	}

	// Тест с большим количеством полей
	largeStruct := mem.AllocateStruct(1000)
	for i := 0; i < 1000; i++ {
		value := mem.GetFieldValue(largeStruct, i)
		if value.String() != "0" {
			t.Errorf("Large struct field %d should be 0", i)
		}
	}

	// Тест с большим массивом
	largeArray := mem.AllocateArray(1000)
	for i := 0; i < 1000; i++ {
		value := mem.GetFromArray(largeArray, i)
		if value.String() != "0" {
			t.Errorf("Large array element %d should be 0", i)
		}
	}
}

// TestZ3AliasingVerification тестирует алиасинг с использованием Z3
func TestZ3AliasingVerification(t *testing.T) {
	mem := NewSymbolicMemory()
	z3Translator := translator.NewZ3Translator()
	defer z3Translator.Close()

	struct1 := mem.AllocateStruct(1)
	struct2 := mem.AllocateStruct(1)

	mem.AssignField(struct1, 0, symbolic.NewIntConstant(10))
	mem.AssignField(struct2, 0, symbolic.NewIntConstant(20))

	alias := mem.CreateAlias(struct1, 100)

	mem.AssignField(alias, 0, symbolic.NewIntConstant(30))

	// Создаем условия для проверки
	// Условие 1: значение в struct1 должно быть 30 (из-за алиасинга)
	condition1 := symbolic.NewBinaryOperation(
		mem.GetFieldValue(struct1, 0),
		symbolic.NewIntConstant(30),
		symbolic.EQ,
	)

	// Условие 2: значение в struct2 должно остаться 20 (не затронуто алиасингом)
	condition2 := symbolic.NewBinaryOperation(
		mem.GetFieldValue(struct2, 0),
		symbolic.NewIntConstant(20),
		symbolic.EQ,
	)

	// Общее условие: оба должны быть истинны
	combinedCondition := symbolic.NewLogicalOperation(
		[]symbolic.SymbolicExpression{condition1, condition2},
		symbolic.AND,
	)

	z3Condition, err := z3Translator.TranslateExpression(combinedCondition)
	if err != nil {
		t.Fatalf("Translation failed: %v", err)
	}

	ctx := z3Translator.GetContext().(*z3.Context)
	solver := z3.NewSolver(ctx)
	solver.Assert(z3Condition.(z3.Bool))

	result, err := solver.Check()
	if err != nil {
		t.Fatalf("Error checking satisfiability: %v", err)
	}
	if !result {
		t.Errorf("Expected condition to be satisfiable, but got: %v", result)
		t.Logf("Memory state: %s", mem.String())
	}

	model := solver.Model()
	t.Logf("Z3 model: %s", model.String())
}

// TestZ3ArrayAliasing тестирует алиасинг массивов с верификацией в Z3
func TestZ3ArrayAliasing(t *testing.T) {
	mem := NewSymbolicMemory()
	z3Translator := translator.NewZ3Translator()
	defer z3Translator.Close()

	arr := mem.AllocateArray(3)

	for i := 0; i < 3; i++ {
		mem.AssignToArray(arr, i, symbolic.NewIntConstant(int64(i*10)))
	}

	alias := mem.CreateAlias(arr, 200)

	mem.AssignToArray(alias, 1, symbolic.NewIntConstant(999))

	conditions := []symbolic.SymbolicExpression{}

	// Проверяем, что изменения через алиас видны в оригинале
	for i := 0; i < 3; i++ {
		var expected symbolic.SymbolicExpression
		if i == 1 {
			expected = symbolic.NewIntConstant(999) // Измененный через алиас
		} else {
			expected = symbolic.NewIntConstant(int64(i * 10)) // Оригинальное значение
		}

		condition := symbolic.NewBinaryOperation(
			mem.GetFromArray(arr, i),
			expected,
			symbolic.EQ,
		)
		conditions = append(conditions, condition)
	}

	combinedCondition := symbolic.NewLogicalOperation(conditions, symbolic.AND)

	z3Condition, err := z3Translator.TranslateExpression(combinedCondition)
	if err != nil {
		t.Fatalf("Translation failed: %v", err)
	}

	ctx := z3Translator.GetContext().(*z3.Context)
	solver := z3.NewSolver(ctx)
	solver.Assert(z3Condition.(z3.Bool))

	result, err := solver.Check()
	if err != nil {
		t.Fatalf("Error checking satisfiability: %v", err)
	}
	if !result {
		t.Errorf("Array aliasing verification failed: %v", result)
	}

	t.Logf("Array aliasing verified successfully with Z3")
}

// TestZ3ComplexAliasingScenario тестирует сценарий алиасинга с Z3
func TestZ3ComplexAliasingScenario(t *testing.T) {
	mem := NewSymbolicMemory()
	z3Translator := translator.NewZ3Translator()
	defer z3Translator.Close()

	person := mem.AllocateStruct(2)
	mem.AssignField(person, 0, symbolic.NewSymbolicVariable("name", symbolic.IntType))
	mem.AssignField(person, 1, symbolic.NewIntConstant(25))

	alias1 := mem.CreateAlias(person, 101)
	alias2 := mem.CreateAlias(person, 102)

	mem.AssignField(alias1, 1, symbolic.NewIntConstant(30))

	// Проверяем, что изменение видно через все ссылки
	conditions := []symbolic.SymbolicExpression{}

	// Проверяем через оригинал
	cond1 := symbolic.NewBinaryOperation(
		mem.GetFieldValue(person, 1),
		symbolic.NewIntConstant(30),
		symbolic.EQ,
	)
	conditions = append(conditions, cond1)

	// Проверяем через alias1
	cond2 := symbolic.NewBinaryOperation(
		mem.GetFieldValue(alias1, 1),
		symbolic.NewIntConstant(30),
		symbolic.EQ,
	)
	conditions = append(conditions, cond2)

	// Проверяем через alias2
	cond3 := symbolic.NewBinaryOperation(
		mem.GetFieldValue(alias2, 1),
		symbolic.NewIntConstant(30),
		symbolic.EQ,
	)
	conditions = append(conditions, cond3)

	combinedCondition := symbolic.NewLogicalOperation(conditions, symbolic.AND)

	z3Condition, err := z3Translator.TranslateExpression(combinedCondition)
	if err != nil {
		t.Fatalf("Translation failed: %v", err)
	}

	ctx := z3Translator.GetContext().(*z3.Context)
	solver := z3.NewSolver(ctx)
	solver.Assert(z3Condition.(z3.Bool))

	result, err := solver.Check()
	if err != nil {
		t.Fatalf("Error checking satisfiability: %v", err)
	}
	if !result {
		t.Errorf("Complex aliasing scenario verification failed: %v", result)
	}

	t.Logf("Complex aliasing scenario verified with Z3")
}

// TestZ3NonAliasingScenario тестирует сценарий без алиасинга с Z3
func TestZ3NonAliasingScenario(t *testing.T) {
	mem := NewSymbolicMemory()
	z3Translator := translator.NewZ3Translator()
	defer z3Translator.Close()

	struct1 := mem.AllocateStruct(1)
	struct2 := mem.AllocateStruct(1)

	mem.AssignField(struct1, 0, symbolic.NewIntConstant(100))
	mem.AssignField(struct2, 0, symbolic.NewIntConstant(200))

	mem.AssignField(struct1, 0, symbolic.NewIntConstant(150))

	// Создаем условия: struct1 изменилась, struct2 осталась прежней
	condition1 := symbolic.NewBinaryOperation(
		mem.GetFieldValue(struct1, 0),
		symbolic.NewIntConstant(150),
		symbolic.EQ,
	)

	condition2 := symbolic.NewBinaryOperation(
		mem.GetFieldValue(struct2, 0),
		symbolic.NewIntConstant(200),
		symbolic.EQ,
	)

	combinedCondition := symbolic.NewLogicalOperation(
		[]symbolic.SymbolicExpression{condition1, condition2},
		symbolic.AND,
	)

	z3Condition, err := z3Translator.TranslateExpression(combinedCondition)
	if err != nil {
		t.Fatalf("Translation failed: %v", err)
	}

	ctx := z3Translator.GetContext().(*z3.Context)
	solver := z3.NewSolver(ctx)
	solver.Assert(z3Condition.(z3.Bool))

	result, err := solver.Check()
	if err != nil {
		t.Fatalf("Error checking satisfiability: %v", err)
	}
	if !result {
		t.Errorf("Non-aliasing scenario verification failed: %v", result)
	}

	t.Logf("Non-aliasing scenario verified with Z3")
}

// TestZ3SymbolicAliasingConditions тестирует символьные условия алиасинга
func TestZ3SymbolicAliasingConditions(t *testing.T) {
	mem := NewSymbolicMemory()
	z3Translator := translator.NewZ3Translator()
	defer z3Translator.Close()

	x := symbolic.NewSymbolicVariable("x", symbolic.IntType)
	y := symbolic.NewSymbolicVariable("y", symbolic.IntType)

	a := mem.AllocateStruct(1)
	b := mem.AllocateStruct(1)

	mem.AssignField(a, 0, x)
	mem.AssignField(b, 0, y)

	alias := mem.CreateAlias(a, 300)

	mem.AssignField(alias, 0, symbolic.NewIntConstant(42))

	// Создаем условие:
	// Если x == y, то после изменения через алиас, b[0] также должно стать 42
	// Но в нашей модели a и b - разные объекты, поэтому b[0] должно остаться y

	// Условие 1: значение в a[0] стало 42 (из-за алиасинга)
	cond1 := symbolic.NewBinaryOperation(
		mem.GetFieldValue(a, 0),
		symbolic.NewIntConstant(42),
		symbolic.EQ,
	)

	// Условие 2: значение в b[0] осталось y (не затронуто)
	cond2 := symbolic.NewBinaryOperation(
		mem.GetFieldValue(b, 0),
		y,
		symbolic.EQ,
	)

	combinedCondition := symbolic.NewLogicalOperation(
		[]symbolic.SymbolicExpression{cond1, cond2},
		symbolic.AND,
	)

	z3Condition, err := z3Translator.TranslateExpression(combinedCondition)
	if err != nil {
		t.Fatalf("Translation failed: %v", err)
	}

	ctx := z3Translator.GetContext().(*z3.Context)
	solver := z3.NewSolver(ctx)
	solver.Assert(z3Condition.(z3.Bool))

	result, err := solver.Check()
	if err != nil {
		t.Fatalf("Error checking satisfiability: %v", err)
	}
	if !result {
		t.Errorf("Symbolic aliasing conditions verification failed: %v", result)
	}

	// Дополнительно: проверяем, что можем найти конкретные значения для x и y
	model := solver.Model()

	xZ3 := z3Translator.VisitVariable(x).(z3.Int)
	yZ3 := z3Translator.VisitVariable(y).(z3.Int)

	xVal := model.Eval(xZ3, true)
	yVal := model.Eval(yZ3, true)

	t.Logf("Model for symbolic aliasing: x=%s, y=%s", xVal.String(), yVal.String())
}

func TestZ3MemoryModelConsistency(t *testing.T) {
	mem := NewSymbolicMemory()
	z3Translator := translator.NewZ3Translator()
	defer z3Translator.Close()

	// Создаем массив структур
	people := mem.AllocateArray(2)

	for i := 0; i < 2; i++ {
		person := mem.AllocateStruct(2)
		mem.AssignField(person, 0, symbolic.NewSymbolicVariable("name", symbolic.IntType))
		mem.AssignField(person, 1, symbolic.NewIntConstant(int64(20+i*5))) // Age: 20, 25

		mem.AssignToArray(people, i, symbolic.NewRef(person.ID, symbolic.StructType))
	}

	// Создаем алиас для первого элемента
	firstPerson := mem.GetFromArray(people, 0).(*symbolic.Ref)
	alias := mem.CreateAlias(firstPerson, 400)

	// Изменяем возраст через алиас
	mem.AssignField(alias, 1, symbolic.NewIntConstant(35))

	// Создаем условия для проверки целостности модели памяти
	conditions := []symbolic.SymbolicExpression{}

	// Проверяем, что изменение видно через оригинальную ссылку
	cond1 := symbolic.NewBinaryOperation(
		mem.GetFieldValue(firstPerson, 1),
		symbolic.NewIntConstant(35),
		symbolic.EQ,
	)
	conditions = append(conditions, cond1)

	// Проверяем, что второй элемент не изменился
	secondPerson := mem.GetFromArray(people, 1).(*symbolic.Ref)
	cond2 := symbolic.NewBinaryOperation(
		mem.GetFieldValue(secondPerson, 1),
		symbolic.NewIntConstant(25),
		symbolic.EQ,
	)
	conditions = append(conditions, cond2)

	combinedCondition := symbolic.NewLogicalOperation(conditions, symbolic.AND)

	z3Condition, err := z3Translator.TranslateExpression(combinedCondition)
	if err != nil {
		t.Fatalf("Translation failed: %v", err)
	}

	ctx := z3Translator.GetContext().(*z3.Context)
	solver := z3.NewSolver(ctx)
	solver.Assert(z3Condition.(z3.Bool))

	result, err := solver.Check()
	if err != nil {
		t.Fatalf("Error checking satisfiability: %v", err)
	}
	if !result {
		t.Errorf("Memory model consistency verification failed: %v", result)
	}

	t.Logf("Memory model consistency verified with Z3")
	t.Logf("Final memory state: %s", mem.String())
}
