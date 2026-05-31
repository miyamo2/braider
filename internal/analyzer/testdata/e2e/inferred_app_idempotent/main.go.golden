package main

// Existing inferred bootstrap with a hash that matches the (empty) dependency graph.
// The analyzer must detect this as current and emit no diagnostic.

func main() {
	_ = dependency
}

// braider:hash:e3b0c44298fc1c14
var dependency = func() struct {
} {
	return struct {
	}{}
}()
