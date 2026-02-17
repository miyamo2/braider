package annotation

type Variable interface {
	_IsVariable()
}

type VariableOption interface {
	_IsVariableOption()
}

type VariableDefault interface {
	_IsVariableDefault()
}

type VariableTyped interface {
	_IsVariableTyped()
}

type VariableNamed interface {
	_IsVariableNamed()
}
