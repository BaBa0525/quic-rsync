package internal

func Unwrap(err error) {
	if err != nil {
		panic(err)
	}
}
