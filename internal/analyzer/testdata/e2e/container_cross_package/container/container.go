package container

import "container_cross_package/service"

type AppContainer struct {
	Svc *service.UserService
}
