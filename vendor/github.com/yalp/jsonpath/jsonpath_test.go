package jsonpath

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

var goessner = []byte(`{
    "store": {
        "book": [
            {
                "category": "reference",
                "author": "Nigel Rees",
                "title": "Sayings of the Century",
                "price": 8.95
            },
            {
                "category": "fiction",
                "author": "Evelyn Waugh",
                "title": "Sword of Honour",
                "price": 12.99
            },
            {
                "category": "fiction",
                "author": "Herman Melville",
                "title": "Moby Dick",
                "isbn": "0-553-21311-3",
                "price": 8.99
            },
            {
                "category": "fiction",
                "author": "J. R. R. Tolkien",
                "title": "The Lord of the Rings",
                "isbn": "0-395-19395-8",
                "price": 22.99
            }
        ],
        "bicycle": {
            "color": "red",
            "price": 19.95
        }
    },
    "expensive": 10
}`)

var sample = map[string]interface{}{
	"A": []interface{}{
		"string",
		23.3,
		3,
		true,
		false,
		nil,
	},
	"B": "value",
	"C": 3.14,
	"D": map[string]interface{}{
		"C": 3.1415,
		"V": []interface{}{
			"string2",
			map[string]interface{}{
				"C": 3.141592,
			},
		},
	},
}

func TestGossner(t *testing.T) {
	var data interface{}
	json.Unmarshal(goessner, &data)

	tests := map[string]interface{}{
		"$.store.book[*].author": []interface{}{"Nigel Rees", "Evelyn Waugh", "Herman Melville", "J. R. R. Tolkien"},
		"$..author":              []interface{}{"Nigel Rees", "Evelyn Waugh", "Herman Melville", "J. R. R. Tolkien"},
	}
	assert(t, data, tests)
}

func TestParsing(t *testing.T) {

	tests := map[string]interface{}{
		"$":          sample,
		"$.A[0]":     "string",
		`$["A"][0]`:  "string",
		`$[A][0]`:    "string",
		"$.A":        []interface{}{"string", 23.3, 3, true, false, nil},
		"$.A[*]":     []interface{}{"string", 23.3, 3, true, false, nil},
		"$.A.*":      []interface{}{"string", 23.3, 3, true, false, nil},
		"$.A.*.a":    []interface{}{},
		"$.*.V[0]":   []interface{}{"string2"},
		"$[B,C]":     []interface{}{"value", 3.14},
		"$.A[1,4,2]": []interface{}{23.3, false, 3},
		"$[C,B]":     []interface{}{3.14, "value"},
		"$.A[1:4]":   []interface{}{23.3, 3, true},
		"$.A[::2]":   []interface{}{"string", 3, false},
		"$.A[-2:]":   []interface{}{false, nil},
		"$.A[:-1]":   []interface{}{"string", 23.3, 3, true, false},
		"$.A[::-1]":  []interface{}{nil, false, true, 3, 23.3, "string"},
		"$..C":       []interface{}{3.1415, 3.141592},
		"$..V[1].C":  []interface{}{3.141592},
		"$..[C]":     []interface{}{3.1415, 3.141592},
		"$.D.V..*": []interface{}{
			"string2",
			map[string]interface{}{
				"C": 3.141592,
			},
			3.141592,
		},
		"$..[0]": []interface{}{"string", "string2"},
		"$..ZZ":  []interface{}{},
	}
	assert(t, sample, tests)
}

func TestErrors(t *testing.T) {
	tests := map[string]string{
		".A":           "path must start with a '$'",
		"$.":           "expected JSON child identifier after '.'",
		"$.1":          "unexcepted token .1",
		"$.A[]":        "expected at least one key, index or expression",
		`$["]`:         "bad string invalid syntax",
		`$[A][0`:       "unexpected end of path",
		"$.ZZZ":        "child 'ZZZ' not found in JSON object",
		"$.A*]":        "unexcepted token *",
		"$.*V":         "unexcepted token V",
		"$[B,C":        "unexpected end of path",
		"$.A[1,4.2]":   "unexpected token '.'",
		"$[C:B]":       "expected JSON array",
		"$.A[1:4:0:0]": "bad range syntax [start:end:step]",
		"$.A[:,]":      "unexpected token ','",
		"$..":          "cannot end with a scan '..'",
		"$..1":         "unexpected token '1' after deep search '..'",
	}
	assertError(t, sample, tests)
}

func assert(t *testing.T, json interface{}, tests map[string]interface{}) {
	for path, expected := range tests {
		actual, err := Read(json, path)
		if err != nil {
			t.Error("failed:", path, err)
		} else if !reflect.DeepEqual(actual, expected) {
			t.Errorf("failed: mismatch for %s\nexpected: %+v\nactual: %+v", path, expected, actual)
		}
	}
}

func assertError(t *testing.T, json interface{}, tests map[string]string) {
	for path, expectedError := range tests {
		_, err := Read(json, path)
		if err == nil {
			t.Error("path", path, "should fail with", expectedError)
		} else if !strings.Contains(err.Error(), expectedError) {
			t.Error("path", path, "shoud fail with ", expectedError, "but failed with:", err)
		}
	}
}
