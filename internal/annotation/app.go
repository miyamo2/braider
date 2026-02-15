package annotation

type AppMarker struct{}

type App interface {
	_IsApp() AppMarker
}

type AppOptionMarker struct{}

type AppOption interface {
	_IsAppOption() AppOptionMarker
}

type AppDefaultMarker struct{}

type AppDefault interface {
	_IsAppDefault() AppDefaultMarker
}

type AppContainerMarker struct{}

type AppContainer interface {
	_IsAppContainerMarker() AppContainerMarker
}
