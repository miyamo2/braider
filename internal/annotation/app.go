package annotation

type App interface {
	_IsApp()
}

type AppOption interface {
	_IsAppOption()
}

type AppDefault interface {
	_IsAppDefault()
}

type AppContainer interface {
	_IsAppContainerMarker()
}
