package main

type InvalidName struct{}

func (InvalidName) Name(prefix string) string { return prefix + "name" }

func main() {}
