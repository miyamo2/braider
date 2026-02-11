package domain

type IRepository interface {
	FindByID(id string) (string, error)
}
