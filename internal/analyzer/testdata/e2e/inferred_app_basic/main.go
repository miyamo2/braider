package main

// No annotation.App is declared. With the optional-app-annotation feature,
// the analyzer infers this single main package as the entry point and generates
// a default-mode bootstrap IIFE wiring the service.

func main() { // want "bootstrap code is missing \\(entry point inferred from single main package; add annotation.App to declare it explicitly\\)"
}
