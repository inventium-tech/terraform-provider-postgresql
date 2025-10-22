package helpers

import (
	"github.com/go-playground/validator/v10"
	"sync"
)

var (
	safeValidator       *validator.Validate
	unsafeValidator     *validator.Validate
	safeValidatorOnce   sync.Once
	unsafeValidatorOnce sync.Once
)

func GetSafeValidator() *validator.Validate {
	safeValidatorOnce.Do(func() {
		safeValidator = validator.New(validator.WithRequiredStructEnabled())
	})
	return safeValidator
}

func GetUnsafeValidator() *validator.Validate {
	unsafeValidatorOnce.Do(func() {
		unsafeValidator = validator.New(validator.WithPrivateFieldValidation())
	})
	return unsafeValidator
}
