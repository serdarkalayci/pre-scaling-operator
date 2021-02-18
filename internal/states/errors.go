package states

type TooMany struct {
	msg   string
	count int
}

func (err TooMany) Error() string {
	return err.msg
}

func (err TooMany) Count() int {
	return err.count
}

type NotFound struct {
	msg string
}

func (err NotFound) Error() string {
	return err.msg
}
