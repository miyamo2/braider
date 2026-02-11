package external

type ExternalNamer struct{}

func (ExternalNamer) Name() string {
	return "externalName"
}
