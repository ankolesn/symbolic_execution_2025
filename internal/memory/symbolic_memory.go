package memory

import (
	"fmt"
	"symbolic-execution-course/internal/symbolic"
)

type Memory interface {
	Allocate(tpe symbolic.ExpressionType) *symbolic.Ref

	AssignField(ref *symbolic.Ref, fieldIdx int, value symbolic.SymbolicExpression)

	GetFieldValue(ref *symbolic.Ref, fieldIdx int) symbolic.SymbolicExpression

	AssignToArray(ref *symbolic.Ref, index int, value symbolic.SymbolicExpression)

	GetFromArray(ref *symbolic.Ref, index int) symbolic.SymbolicExpression

	AllocateStruct(fieldCount int) *symbolic.Ref
	AllocateArray(length int) *symbolic.Ref
}

type SymbolicMemory struct {
	objects      map[int]*MemoryObject
	nextObjectID int
	aliases      map[int]int // map[aliasID]originalID
}

type MemoryObject struct {
	Type   symbolic.ExpressionType
	Fields map[int]symbolic.SymbolicExpression // для структур
	Elems  map[int]symbolic.SymbolicExpression // для массивов
}

func NewSymbolicMemory() *SymbolicMemory {
	return &SymbolicMemory{
		objects:      make(map[int]*MemoryObject),
		nextObjectID: 1,
		aliases:      make(map[int]int),
	}
}

func (sm *SymbolicMemory) Allocate(tpe symbolic.ExpressionType) *symbolic.Ref {
	id := sm.nextObjectID
	sm.nextObjectID++

	sm.objects[id] = &MemoryObject{
		Type:   tpe,
		Fields: make(map[int]symbolic.SymbolicExpression),
		Elems:  make(map[int]symbolic.SymbolicExpression),
	}

	return symbolic.NewRef(id, tpe)
}

func (sm *SymbolicMemory) getOriginalID(ref *symbolic.Ref) int {
	if originalID, exists := sm.aliases[ref.ID]; exists {
		return originalID
	}
	return ref.ID
}

func (sm *SymbolicMemory) AssignField(ref *symbolic.Ref, fieldIdx int, value symbolic.SymbolicExpression) {
	originalID := sm.getOriginalID(ref)
	obj, exists := sm.objects[originalID]
	if !exists {
		panic(fmt.Sprintf("Объект с ID %d не найден", originalID))
	}

	if obj.Type != symbolic.StructType {
		panic("Попытка присвоить поле не-структуре")
	}

	obj.Fields[fieldIdx] = value
}

func (sm *SymbolicMemory) GetFieldValue(ref *symbolic.Ref, fieldIdx int) symbolic.SymbolicExpression {
	originalID := sm.getOriginalID(ref)
	obj, exists := sm.objects[originalID]
	if !exists {
		panic(fmt.Sprintf("Объект с ID %d не найден", originalID))
	}

	if obj.Type != symbolic.StructType {
		panic("Попытка прочитать поле не-структуры")
	}

	value, exists := obj.Fields[fieldIdx]
	if !exists {
		return symbolic.NewIntConstant(0)
	}

	return value
}

func (sm *SymbolicMemory) AssignToArray(ref *symbolic.Ref, index int, value symbolic.SymbolicExpression) {
	originalID := sm.getOriginalID(ref)
	obj, exists := sm.objects[originalID]
	if !exists {
		panic(fmt.Sprintf("Объект с ID %d не найден", originalID))
	}

	if obj.Type != symbolic.ArrayType {
		panic("Попытка присвоить элемент не-массиву")
	}

	obj.Elems[index] = value
}

func (sm *SymbolicMemory) GetFromArray(ref *symbolic.Ref, index int) symbolic.SymbolicExpression {
	originalID := sm.getOriginalID(ref)
	obj, exists := sm.objects[originalID]
	if !exists {
		panic(fmt.Sprintf("Объект с ID %d не найден", originalID))
	}

	if obj.Type != symbolic.ArrayType {
		panic("Попытка прочитать элемент не-массива")
	}

	value, exists := obj.Elems[index]
	if !exists {
		return symbolic.NewIntConstant(0)
	}

	return value
}

// CreateAlias создаёт алиас для существующей ссылки
func (sm *SymbolicMemory) CreateAlias(original *symbolic.Ref, aliasID int) *symbolic.Ref {
	originalID := sm.getOriginalID(original)
	sm.aliases[aliasID] = originalID
	return symbolic.NewRef(aliasID, original.ExprType)
}

// String возвращает строковое представление состояния памяти
func (sm *SymbolicMemory) String() string {
	result := "Symbolic Memory State:\n"

	for id, obj := range sm.objects {
		result += fmt.Sprintf("  Object %d (%s):\n", id, obj.Type.String())

		switch obj.Type {
		case symbolic.StructType:
			for fieldIdx, field := range obj.Fields {
				result += fmt.Sprintf("    Field[%d]: %s\n", fieldIdx, field.String())
			}
		case symbolic.ArrayType:
			for index, elem := range obj.Elems {
				result += fmt.Sprintf("    Elem[%d]: %s\n", index, elem.String())
			}
		default:
			result += fmt.Sprintf("    Simple type: %s\n", obj.Type.String())
		}
	}

	result += "Aliases:\n"
	for alias, original := range sm.aliases {
		result += fmt.Sprintf("  %d -> %d\n", alias, original)
	}

	return result
}

// AllocateStruct создает структуру с заданным количеством полей
func (sm *SymbolicMemory) AllocateStruct(fieldCount int) *symbolic.Ref {
	id := sm.nextObjectID
	sm.nextObjectID++

	obj := &MemoryObject{
		Type:   symbolic.StructType,
		Fields: make(map[int]symbolic.SymbolicExpression),
		Elems:  make(map[int]symbolic.SymbolicExpression),
	}

	for i := 0; i < fieldCount; i++ {
		obj.Fields[i] = symbolic.NewIntConstant(0)
	}

	sm.objects[id] = obj
	return symbolic.NewRef(id, symbolic.StructType)
}

// AllocateArray создает массив заданной длины
func (sm *SymbolicMemory) AllocateArray(length int) *symbolic.Ref {
	id := sm.nextObjectID
	sm.nextObjectID++

	obj := &MemoryObject{
		Type:   symbolic.ArrayType,
		Fields: make(map[int]symbolic.SymbolicExpression),
		Elems:  make(map[int]symbolic.SymbolicExpression),
	}

	for i := 0; i < length; i++ {
		obj.Elems[i] = symbolic.NewIntConstant(0)
	}

	sm.objects[id] = obj
	return symbolic.NewRef(id, symbolic.ArrayType)
}
