package annotation

type AppMarker struct{}

type App interface {
	_IsApp() AppMarker
}
