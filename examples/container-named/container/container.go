// Package container defines the application's dependency container.
package container

import "github.com/miyamo2/braider/examples/container-named/service"

// AppContainer is a user-defined container struct.
// Its fields are resolved by the braider analyzer against registered dependencies.
// Fields without braider struct tags are matched by type.
type AppContainer struct {
	Svc *service.UserService
}
