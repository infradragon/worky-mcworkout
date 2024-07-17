package urit

type TemplateParseError interface {
	error
	Unwrap() error
	Position() int
}

func newTemplateParseError(msg string, pos int, err error) TemplateParseError {
	return &templateParseError{
		msg: msg,
		err: err,
		pos: pos,
	}
}

type templateParseError struct {
	msg string
	err error
	pos int
}

func (e *templateParseError) Error() string {
	return e.msg
}

func (e *templateParseError) Unwrap() error {
	return e.err
}

func (e *templateParseError) Position() int {
	return e.pos
}
