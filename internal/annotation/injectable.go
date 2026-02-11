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

type InjectableTyped[T any] interface {
	_IsInjectableTyped() T
}

type InjectableNamed[T Namer] interface {
	_IsInjectableNamed() T
}

type InjectableWithoutConstructorMarker struct{}

type InjectableWithoutConstructor interface {
	_IsInjectableWithoutConstructor() InjectableWithoutConstructorMarker
}
