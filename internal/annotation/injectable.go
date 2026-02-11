package annotation

type InjectableMarker struct{}

type Injectable interface {
	_IsInjectable() InjectableMarker
}

type InjectableOptionMarker struct{}

type InjectableOption interface {
	_IsInjectableOption() InjectableOptionMarker
}

type InjectableDefaultMarker struct{}

type InjectableDefault interface {
	_IsInjectableDefault() InjectableDefaultMarker
}

type InjectableTypedMarker struct{}

type InjectableTyped interface {
	_IsInjectableTyped() InjectableTypedMarker
}

type InjectableNamedMarker struct{}

type InjectableNamed interface {
	_IsInjectableNamed() InjectableNamedMarker
}

type InjectableWithoutConstructorMarker struct{}

type InjectableWithoutConstructor interface {
	_IsInjectableWithoutConstructor() InjectableWithoutConstructorMarker
}
