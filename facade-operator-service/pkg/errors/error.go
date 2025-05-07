package customerrors

type ExpectedError struct {
	Message string
}

func (e *ExpectedError) Error() string {
	return e.Message
}
