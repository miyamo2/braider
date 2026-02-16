package main

import "container_named/service"

type MyContainer struct {
	Svc *service.UserService
}
