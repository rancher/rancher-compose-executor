package convert

type errDependencyNotFound struct {
	err error
}

func (e errDependencyNotFound) Error() string {
	return e.err.Error()
}

func NewErrDependencyNotFound(err error) error {
	return errDependencyNotFound{err}
}

func IsErrDependencyNotFound(err error) bool {
	_, ok := err.(errDependencyNotFound)
	return ok
}
