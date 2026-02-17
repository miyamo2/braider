// Package service contains the application's service layer.
package service

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

// UserService is registered as a DI dependency.
type UserService struct {
	annotation.Injectable[inject.Default]
}
