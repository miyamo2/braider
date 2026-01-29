package missing_constructor

import "github.com/miyamo2/braider/pkg/annotation"

// UserRepository is a Provide struct without a constructor.
type UserRepository struct { // want "Provide struct UserRepository requires a constructor"
	annotation.Provide
}
