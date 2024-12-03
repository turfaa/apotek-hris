package validatorx

import (
	"github.com/go-playground/validator/v10"
	dv "github.com/sblackstone/shopspring-decimal-validators"
)

type ValidationErrors = validator.ValidationErrors

var v = validator.New(validator.WithRequiredStructEnabled())

func init() {
	dv.RegisterDecimalValidators(v)
}

func Validate(i any) error {
	return v.Struct(i)
}
