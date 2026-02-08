package main

type MultiReturn struct{}

func (MultiReturn) Name() (string, error) { return "name", nil }

func main() {}
