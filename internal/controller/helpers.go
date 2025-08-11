package controller

import "fmt"

func handleError(err error) {
	if err != nil {
		panic(fmt.Sprintf("Something went wrong: %v", err))
	}
}
