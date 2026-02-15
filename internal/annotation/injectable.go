package annotation

type Injectable interface {
	_IsInjectable()
}

type InjectableOption interface {
	_IsInjectableOption()
}

type InjectableDefault interface {
	_IsInjectableDefault()
}

type InjectableTyped interface {
	_IsInjectableTyped()
}

type InjectableNamed interface {
	_IsInjectableNamed()
}

type InjectableWithoutConstructor interface {
	_IsInjectableWithoutConstructor()
}
