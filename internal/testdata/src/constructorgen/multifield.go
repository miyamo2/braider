package constructorgen

import "github.com/miyamo2/braider/pkg/annotation"

type OrderService struct { // want "missing constructor for OrderService"
	annotation.Inject
	repo   OrderRepository
	logger Logger
	config Config
}

type OrderRepository interface {
	Save(order *Order) error
}

type Logger interface {
	Info(msg string)
}

type Config struct {
	Debug bool
}

type Order struct {
	ID string
}
