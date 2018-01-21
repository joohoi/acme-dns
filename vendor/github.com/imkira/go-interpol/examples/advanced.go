package main

import (
	"io"
	"log"
	"os"
	"strings"

	"github.com/imkira/go-interpol"
)

func main() {
	template, err := os.Open("template.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer template.Close()
	output, err := os.Create("output.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer output.Close()
	opts := &interpol.Options{
		Template: template,
		Format:   toUpper,
		Output:   output,
	}
	i := interpol.NewWithOptions(opts)
	if err := i.Interpolate(); err != nil {
		log.Fatal(err)
	}
}

func toUpper(key string, w io.Writer) error {
	_, err := w.Write([]byte(strings.ToUpper(key)))
	return err
}
