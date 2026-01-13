package parser

import "fmt"

type ClientError struct {
	message string
}

func (e ClientError) Error() string {
	return fmt.Sprintf("[Client error]: %s", e.message)
}

type ServerError struct {
	message string
}

func (e ServerError) Error() string {
	return fmt.Sprintf("[Server error]: %s", e.message)
}
