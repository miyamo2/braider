package annotation

type Provider interface {
	_IsProvider()
}

type ProviderOption interface {
	_IsProviderOption()
}

type ProviderDefault interface {
	_IsProviderDefault()
}

type ProviderTyped interface {
	_IsProviderTyped()
}

type ProviderNamed interface {
	_IsProviderNamed()
}
