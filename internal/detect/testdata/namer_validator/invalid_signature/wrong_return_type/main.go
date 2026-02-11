package main

type WrongType struct{}

func (WrongType) Name() int { return 42 }

func main() {}
