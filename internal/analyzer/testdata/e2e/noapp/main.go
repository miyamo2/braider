package main

// This package has no explicit App annotation. With the optional-app-annotation
// feature, the analyzer infers this single main package as the entry point and
// generates an empty bootstrap IIFE.

func main() { // want "bootstrap code is missing \\(entry point inferred from single main package; add annotation.App to declare it explicitly\\)"
	// No annotation.App call
}
