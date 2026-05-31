package main

// Two main packages in scope with no explicit annotation.App declaration.
// The analyzer must emit an ambiguous-entry-point diagnostic at each main and
// generate no bootstrap.

func main() { // want "multiple main packages in scope and no annotation.App declared; add annotation.App\\[T\\]\\(main\\) to one of: example.com/ambiguous_entry_point/cmd/a, example.com/ambiguous_entry_point/cmd/b"
}
