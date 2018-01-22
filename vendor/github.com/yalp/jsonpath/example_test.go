package jsonpath_test

import (
	"encoding/json"
	"fmt"

	"github.com/yalp/jsonpath"
)

func ExampleRead() {
	raw := []byte(`{"hello":"world"}`)

	var data interface{}
	json.Unmarshal(raw, &data)

	out, err := jsonpath.Read(data, "$.hello")
	if err != nil {
		panic(err)
	}

	fmt.Print(out)
	// Output: world
}

func ExamplePrepare() {
	raw := []byte(`{"hello":"world"}`)

	helloFilter, err := jsonpath.Prepare("$.hello")
	if err != nil {
		panic(err)
	}

	var data interface{}
	if err = json.Unmarshal(raw, &data); err != nil {
		panic(err)
	}

	out, err := helloFilter(data)
	if err != nil {
		panic(err)
	}

	fmt.Print(out)
	// Output: world
}
