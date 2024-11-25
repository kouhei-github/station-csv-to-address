package utils

import "fmt"

type CustomError struct {
	Message string
}

func (e *CustomError) Error() string {
	fmt.Println("Error: " + e.Message)
	return e.Message
}
