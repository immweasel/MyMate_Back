package customerror

import "fmt"

type CustomError struct {
	Module   string
	Endpoint string
	Message  string
}

var ErrTimedOut = fmt.Errorf("TimedOut")

var ErrWrongCredentials = fmt.Errorf("WrongCredentials")

var ErrUUIDAlreadyExists = fmt.Errorf("UUID already exists")

var ErrUserAlreadyExists = fmt.Errorf("UserAlreadyExists")

var ErrJwtInvalid = fmt.Errorf("JWTInvalid")

var ErrJwtVersionIncorrect = fmt.Errorf("JwtVersionIncorrect")

var ErrUserAlreadyActivated = fmt.Errorf("UserAlreadyActivated")

var ErrEmailNotSet = fmt.Errorf("EmailNotSet")

var ErrAttemptsEnded = fmt.Errorf("AttemptsEnded")

func (customError CustomError) Error() string {
	return fmt.Sprintf("ERROR|%s|%s:%s", customError.Endpoint, customError.Module, customError.Message)
}

func (customError *CustomError) AppendModule(module string) {
	customError.Module = module + "." + customError.Module
}

func NewError(module, endpoint, message string) error {
	return CustomError{
		Module:   module,
		Endpoint: endpoint,
		Message:  message,
	}
}
