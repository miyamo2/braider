package domain

type IUserRepository interface {
	FindByID(string) (User, error)
}

type User struct {
	ID   string
	Name string
}
