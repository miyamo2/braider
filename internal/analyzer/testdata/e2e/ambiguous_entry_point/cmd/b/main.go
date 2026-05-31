package main

// Second main package; expects the same ambiguity diagnostic listing both paths.

func main() { // want "multiple main packages in scope and no annotation.App declared; add annotation.App\\[T\\]\\(main\\) to one of: example.com/ambiguous_entry_point/cmd/a, example.com/ambiguous_entry_point/cmd/b"
}
