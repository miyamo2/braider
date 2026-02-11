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

type ProviderTypedMarker struct{}

type ProviderTyped interface {
	_IsProviderTyped() ProviderTypedMarker
}

type ProviderNamedMarker struct{}

type ProviderNamed interface {
	_IsProviderNamed() ProviderNamedMarker
}
