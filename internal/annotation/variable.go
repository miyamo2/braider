package annotation

type VariableMarker struct{}

type Variable interface {
	_IsVariable() VariableMarker
}

type VariableOptionMarker struct{}

type VariableOption interface {
	_IsVariableOption() VariableOptionMarker
}

type VariableDefaultMarker struct{}

type VariableDefault interface {
	_IsVariableDefault() VariableDefaultMarker
}

type VariableTypedMarker struct{}

type VariableTyped interface {
	_IsVariableTyped() VariableTypedMarker
}

type VariableNamedMarker struct{}

type VariableNamed interface {
	_IsVariableNamed() VariableNamedMarker
}
