package annotation

type App interface {
	_IsApp()
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
