package main

import (
	"fmt"

	"github.com/imkira/go-interpol"
)

func main() {
	m := map[string]string{
		"foo": "Hello",
		"bar": "World",
	}
	str, err := interpol.WithMap("{foo} {bar}!!!", m)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Println(str)
}
