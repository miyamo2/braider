package example

type Repository struct{}

func NewRepository() *Repository {
	return &Repository{}
}

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

type App struct {
	service *Service
}

func NewApp(service *Service) *App {
	return &App{service: service}
}
