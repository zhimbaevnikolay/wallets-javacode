package errtranslate

import (
	"fmt"

	"github.com/go-playground/validator/v10"
)

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type FieldErrors []string

type ErrorResponse struct {
	Error   string       `json:"error"`
	Details []FieldError `json:"details"`
}

func TranslateValidationErrors(err validator.ValidationErrors) FieldErrors {
	var fieldErrors FieldErrors

	for _, e := range err {
		field := e.Field()
		tag := e.Tag()

		// Создаем человекочитаемые сообщения
		var message string
		switch tag {
		case "required":
			message = fmt.Sprintf("%s is required", field)
		case "uuid4":
			message = fmt.Sprintf("%s must be a valid uuid4", field)
		case "gte":
			message = fmt.Sprintf("%s must be greater than or equal to %s", field, e.Param())
		case "oneof":
			message = fmt.Sprintf("%s must be in (%s)", field, e.Param())

		default:
			message = fmt.Sprintf("Invalid value for field %s", field)
		}

		fieldErrors = append(fieldErrors, message)
	}

	return fieldErrors
}
