// Package annotation provides marker types and functions for braider's
// dependency injection code generation.
//
// This package defines annotations that instruct the braider analyzer
// to generate constructors, wiring code, and main function bootstrapping.
package annotation

// Inject marks a struct as a dependency injection target.
//
// When a struct is annotated with Inject, braider generates a constructor and
// the required wiring code to resolve and provide its dependencies.
type Inject struct{}

// App marks a struct as the top-level dependency injection target.
//
// When a struct is annotated with App, braider generates the main function
// to bootstrap the application by resolving and injecting all dependencies.
//
// Example:
//
//	package main
//
//	import "github.com/miyamo2/braider/pkg/annotation"
//
//	var _ = annotation.App(main)
//
//	func main() {}
//
// Example generated main function:
//
//	package main
//
//	import "github.com/miyamo2/braider/pkg/annotation"
//
//	var _ = annotation.App(main)
//
//	func main() {
//		_ = dependency
//	}
//
//	var dependency = func() struct {
//		myRepository MyRepository
//		myService    MyService
//		myHandler    MyHandler
//	} {
//		myRepository := NewMyRepository()
//		myService := NewMyService(myRepository)
//		myHandler := NewMyHandler(myService)
//		return struct {
//			myRepository MyRepository
//			myService    MyService
//			myHandler    MyHandler
//		} {
//			myRepository: myRepository,
//			myService:    myService,
//			myHandler:    myHandler,
//		}
//	}
func App(_ func()) struct{} {
	return struct{}{}
}
