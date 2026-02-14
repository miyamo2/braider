package domain

// Logger is an interface dependency resolved via interface registry (untagged).
type Logger interface {
	Log(msg string)
}
