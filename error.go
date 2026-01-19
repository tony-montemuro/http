package http

import "fmt"

type ClientError struct {
	message string
	status  int
}

func (e ClientError) Error() string {
	return fmt.Sprintf("[Client error]: %s", e.message)
}

type ServerError struct {
	message string
	status  int
}

func (e ServerError) Error() string {
	return fmt.Sprintf("[Server error]: %s", e.message)
}
