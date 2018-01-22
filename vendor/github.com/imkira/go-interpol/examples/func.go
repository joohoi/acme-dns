package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/imkira/go-interpol"
)

func main() {
	toUpper := func(key string, w io.Writer) error {
		_, err := w.Write([]byte(strings.ToUpper(key)))
		return err
	}
	str, err := interpol.WithFunc("{foo} {bar}!!!", toUpper)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Println(str)
}
