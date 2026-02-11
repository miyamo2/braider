package annotation

type ProviderMarker struct{}

type Provider interface {
	_IsProvider() ProviderMarker
}

type ProviderOptionMarker struct{}

type ProviderOption interface {
	_IsProviderOption() ProviderOptionMarker
}

type ProviderDefaultMarker struct{}

type ProviderDefault interface {
	_IsProviderDefault() ProviderDefaultMarker
}

type ProviderTyped[T any] interface {
	_IsProviderTyped() T
}

type ProviderNamed[T Namer] interface {
	_IsProviderNamed() T
}
