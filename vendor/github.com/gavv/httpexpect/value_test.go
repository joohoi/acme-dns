package httpexpect

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValueFailed(t *testing.T) {
	chain := makeChain(newMockReporter(t))

	chain.fail("fail")

	value := &Value{chain, nil}

	value.chain.assertFailed(t)

	value.Path("$").chain.assertFailed(t)
	value.Schema("")

	assert.False(t, value.Object() == nil)
	assert.False(t, value.Array() == nil)
	assert.False(t, value.String() == nil)
	assert.False(t, value.Number() == nil)
	assert.False(t, value.Boolean() == nil)
	assert.False(t, value.Path("/") == nil)

	value.Object().chain.assertFailed(t)
	value.Array().chain.assertFailed(t)
	value.String().chain.assertFailed(t)
	value.Number().chain.assertFailed(t)
	value.Boolean().chain.assertFailed(t)

	value.Null()
	value.NotNull()

	value.Equal(nil)
	value.NotEqual(nil)
}

func TestValueCastNull(t *testing.T) {
	reporter := newMockReporter(t)

	var data interface{}

	NewValue(reporter, data).Object().chain.assertFailed(t)
	NewValue(reporter, data).Array().chain.assertFailed(t)
	NewValue(reporter, data).String().chain.assertFailed(t)
	NewValue(reporter, data).Number().chain.assertFailed(t)
	NewValue(reporter, data).Boolean().chain.assertFailed(t)
	NewValue(reporter, data).NotNull().chain.assertFailed(t)
	NewValue(reporter, data).Null().chain.assertOK(t)
}

func TestValueCastIndirectNull(t *testing.T) {
	reporter := newMockReporter(t)

	var data []interface{}

	NewValue(reporter, data).Object().chain.assertFailed(t)
	NewValue(reporter, data).Array().chain.assertFailed(t)
	NewValue(reporter, data).String().chain.assertFailed(t)
	NewValue(reporter, data).Number().chain.assertFailed(t)
	NewValue(reporter, data).Boolean().chain.assertFailed(t)
	NewValue(reporter, data).NotNull().chain.assertFailed(t)
	NewValue(reporter, data).Null().chain.assertOK(t)
}

func TestValueCastBad(t *testing.T) {
	reporter := newMockReporter(t)

	data := func() {}

	NewValue(reporter, data).Object().chain.assertFailed(t)
	NewValue(reporter, data).Array().chain.assertFailed(t)
	NewValue(reporter, data).String().chain.assertFailed(t)
	NewValue(reporter, data).Number().chain.assertFailed(t)
	NewValue(reporter, data).Boolean().chain.assertFailed(t)
	NewValue(reporter, data).NotNull().chain.assertFailed(t)
	NewValue(reporter, data).Null().chain.assertFailed(t)
}

func TestValueCastObject(t *testing.T) {
	reporter := newMockReporter(t)

	data := map[string]interface{}{}

	NewValue(reporter, data).Object().chain.assertOK(t)
	NewValue(reporter, data).Array().chain.assertFailed(t)
	NewValue(reporter, data).String().chain.assertFailed(t)
	NewValue(reporter, data).Number().chain.assertFailed(t)
	NewValue(reporter, data).Boolean().chain.assertFailed(t)
	NewValue(reporter, data).NotNull().chain.assertOK(t)
	NewValue(reporter, data).Null().chain.assertFailed(t)
}

func TestValueCastArray(t *testing.T) {
	reporter := newMockReporter(t)

	data := []interface{}{}

	NewValue(reporter, data).Object().chain.assertFailed(t)
	NewValue(reporter, data).Array().chain.assertOK(t)
	NewValue(reporter, data).String().chain.assertFailed(t)
	NewValue(reporter, data).Number().chain.assertFailed(t)
	NewValue(reporter, data).Boolean().chain.assertFailed(t)
	NewValue(reporter, data).NotNull().chain.assertOK(t)
	NewValue(reporter, data).Null().chain.assertFailed(t)
}

func TestValueCastString(t *testing.T) {
	reporter := newMockReporter(t)

	data := ""

	NewValue(reporter, data).Object().chain.assertFailed(t)
	NewValue(reporter, data).Array().chain.assertFailed(t)
	NewValue(reporter, data).String().chain.assertOK(t)
	NewValue(reporter, data).Number().chain.assertFailed(t)
	NewValue(reporter, data).Boolean().chain.assertFailed(t)
	NewValue(reporter, data).NotNull().chain.assertOK(t)
	NewValue(reporter, data).Null().chain.assertFailed(t)
}

func TestValueCastNumber(t *testing.T) {
	reporter := newMockReporter(t)

	data := 0.0

	NewValue(reporter, data).Object().chain.assertFailed(t)
	NewValue(reporter, data).Array().chain.assertFailed(t)
	NewValue(reporter, data).String().chain.assertFailed(t)
	NewValue(reporter, data).Number().chain.assertOK(t)
	NewValue(reporter, data).Boolean().chain.assertFailed(t)
	NewValue(reporter, data).NotNull().chain.assertOK(t)
	NewValue(reporter, data).Null().chain.assertFailed(t)
}

func TestValueCastBoolean(t *testing.T) {
	reporter := newMockReporter(t)

	data := false

	NewValue(reporter, data).Object().chain.assertFailed(t)
	NewValue(reporter, data).Array().chain.assertFailed(t)
	NewValue(reporter, data).String().chain.assertFailed(t)
	NewValue(reporter, data).Number().chain.assertFailed(t)
	NewValue(reporter, data).Boolean().chain.assertOK(t)
	NewValue(reporter, data).NotNull().chain.assertOK(t)
	NewValue(reporter, data).Null().chain.assertFailed(t)
}

func TestValueGetObject(t *testing.T) {
	type (
		myMap map[string]interface{}
	)

	reporter := newMockReporter(t)

	data1 := map[string]interface{}{"foo": 123.0}

	value1 := NewValue(reporter, data1)
	inner1 := value1.Object()

	inner1.chain.assertOK(t)
	inner1.chain.reset()
	assert.Equal(t, data1, inner1.Raw())

	data2 := myMap{"foo": 123.0}

	value2 := NewValue(reporter, data2)
	inner2 := value2.Object()

	inner2.chain.assertOK(t)
	inner2.chain.reset()
	assert.Equal(t, map[string]interface{}(data2), inner2.Raw())
}

func TestValueGetArray(t *testing.T) {
	type (
		myArray []interface{}
	)

	reporter := newMockReporter(t)

	data1 := []interface{}{"foo", 123.0}

	value1 := NewValue(reporter, data1)
	inner1 := value1.Array()

	inner1.chain.assertOK(t)
	inner1.chain.reset()
	assert.Equal(t, data1, inner1.Raw())

	data2 := myArray{"foo", 123.0}

	value2 := NewValue(reporter, data2)
	inner2 := value2.Array()

	inner2.chain.assertOK(t)
	inner2.chain.reset()
	assert.Equal(t, []interface{}(data2), inner2.Raw())
}

func TestValueGetString(t *testing.T) {
	reporter := newMockReporter(t)

	value := NewValue(reporter, "foo")
	inner := value.String()

	inner.chain.assertOK(t)
	inner.chain.reset()
	assert.Equal(t, "foo", inner.Raw())
}

func TestValueGetNumber(t *testing.T) {
	type (
		myInt int
	)

	reporter := newMockReporter(t)

	data1 := 123.0

	value1 := NewValue(reporter, data1)
	inner1 := value1.Number()

	inner1.chain.assertOK(t)
	inner1.chain.reset()
	assert.Equal(t, data1, inner1.Raw())

	data2 := 123

	value2 := NewValue(reporter, data2)
	inner2 := value2.Number()

	inner2.chain.assertOK(t)
	inner2.chain.reset()
	assert.Equal(t, float64(data2), inner2.Raw())

	data3 := myInt(123)

	value3 := NewValue(reporter, data3)
	inner3 := value3.Number()

	inner3.chain.assertOK(t)
	inner3.chain.reset()
	assert.Equal(t, float64(data3), inner3.Raw())
}

func TestValueGetBoolean(t *testing.T) {
	reporter := newMockReporter(t)

	value1 := NewValue(reporter, true)
	inner1 := value1.Boolean()

	inner1.chain.assertOK(t)
	inner1.chain.reset()
	assert.Equal(t, true, inner1.Raw())

	value2 := NewValue(reporter, false)
	inner2 := value2.Boolean()

	inner2.chain.assertOK(t)
	inner2.chain.reset()
	assert.Equal(t, false, inner2.Raw())
}

func TestValueEqual(t *testing.T) {
	reporter := newMockReporter(t)

	data1 := map[string]interface{}{"foo": "bar"}
	data2 := "baz"

	NewValue(reporter, data1).Equal(data1).chain.assertOK(t)
	NewValue(reporter, data2).Equal(data2).chain.assertOK(t)

	NewValue(reporter, data1).NotEqual(data1).chain.assertFailed(t)
	NewValue(reporter, data2).NotEqual(data2).chain.assertFailed(t)

	NewValue(reporter, data1).Equal(data2).chain.assertFailed(t)
	NewValue(reporter, data2).Equal(data1).chain.assertFailed(t)

	NewValue(reporter, data1).NotEqual(data2).chain.assertOK(t)
	NewValue(reporter, data2).NotEqual(data1).chain.assertOK(t)

	NewValue(reporter, nil).Equal(nil).chain.assertOK(t)

	NewValue(reporter, nil).Equal(map[string]interface{}(nil)).chain.assertOK(t)
	NewValue(reporter, nil).Equal(map[string]interface{}{}).chain.assertFailed(t)

	NewValue(reporter, data1).Equal(func() {}).chain.assertFailed(t)
	NewValue(reporter, data1).NotEqual(func() {}).chain.assertFailed(t)
}

func TestValuePathObject(t *testing.T) {
	reporter := newMockReporter(t)

	user0 := map[string]interface{}{"name": "john"}
	user1 := map[string]interface{}{"name": "bob"}

	data := map[string]interface{}{
		"users": []interface{}{
			user0,
			user1,
		},
	}

	value := NewValue(reporter, data)

	assert.Equal(t, data, value.Path("$").Raw())
	assert.Equal(t, data["users"], value.Path("$.users").Raw())
	assert.Equal(t, user0, value.Path("$.users[0]").Raw())
	assert.Equal(t, "john", value.Path("$.users[0].name").Raw())
	assert.Equal(t, []interface{}{"john", "bob"}, value.Path("$.users[*].name").Raw())
	assert.Equal(t, []interface{}{"john", "bob"}, value.Path("$..name").Raw())
	value.chain.assertOK(t)

	names := value.Path("$..name").Array().Iter()
	names[0].String().Equal("john").chain.assertOK(t)
	names[1].String().Equal("bob").chain.assertOK(t)
	value.chain.assertOK(t)

	for _, key := range []string{"$.bad", "!"} {
		bad := value.Path(key)
		assert.True(t, bad != nil)
		assert.True(t, bad.Raw() == nil)
		value.chain.assertFailed(t)
		value.chain.reset()
	}
}

func TestValuePathArray(t *testing.T) {
	reporter := newMockReporter(t)

	user0 := map[string]interface{}{"name": "john"}
	user1 := map[string]interface{}{"name": "bob"}

	data := []interface{}{
		user0,
		user1,
	}

	value := NewValue(reporter, data)

	assert.Equal(t, data, value.Path("$").Raw())
	assert.Equal(t, user0, value.Path("$[0]").Raw())
	assert.Equal(t, "john", value.Path("$[0].name").Raw())
	assert.Equal(t, []interface{}{"john", "bob"}, value.Path("$[*].name").Raw())
	assert.Equal(t, []interface{}{"john", "bob"}, value.Path("$..name").Raw())
	value.chain.assertOK(t)
}

func TestValuePathString(t *testing.T) {
	reporter := newMockReporter(t)

	data := "foo"

	value := NewValue(reporter, data)

	assert.Equal(t, data, value.Path("$").Raw())
	value.chain.assertOK(t)
}

func TestValuePathNumber(t *testing.T) {
	reporter := newMockReporter(t)

	data := 123

	value := NewValue(reporter, data)

	assert.Equal(t, float64(data), value.Path("$").Raw())
	value.chain.assertOK(t)
}

func TestValuePathBoolean(t *testing.T) {
	reporter := newMockReporter(t)

	data := true

	value := NewValue(reporter, data)

	assert.Equal(t, data, value.Path("$").Raw())
	value.chain.assertOK(t)
}

func TestValuePathNull(t *testing.T) {
	reporter := newMockReporter(t)

	value := NewValue(reporter, nil)

	assert.Equal(t, nil, value.Path("$").Raw())
	value.chain.assertOK(t)
}

func TestValuePathError(t *testing.T) {
	reporter := newMockReporter(t)

	data := "foo"

	value := NewValue(reporter, data)

	for _, key := range []string{"$.bad", "!"} {
		bad := value.Path(key)
		assert.True(t, bad != nil)
		assert.True(t, bad.Raw() == nil)
		value.chain.assertFailed(t)
	}
}

// based on github.com/yalp/jsonpath
func TestValuePathExpressions(t *testing.T) {
	data := map[string]interface{}{
		"A": []interface{}{
			"string",
			23.3,
			3.0,
			true,
			false,
			nil,
		},
		"B": "value",
		"C": 3.14,
		"D": map[string]interface{}{
			"C": 3.1415,
			"V": []interface{}{
				"string2a",
				"string2b",
				map[string]interface{}{
					"C": 3.141592,
				},
			},
		},
		"E": map[string]interface{}{
			"A": []interface{}{"string3"},
			"D": map[string]interface{}{
				"V": map[string]interface{}{
					"C": 3.14159265,
				},
			},
		},
		"F": map[string]interface{}{
			"V": []interface{}{
				"string4a",
				"string4b",
				map[string]interface{}{
					"CC": 3.1415926535,
				},
				map[string]interface{}{
					"CC": "hello",
				},
				[]interface{}{
					"string5a",
					"string5b",
				},
				[]interface{}{
					"string6a",
					"string6b",
				},
			},
		},
	}

	reporter := newMockReporter(t)

	runTests := func(tests map[string]interface{}) {
		value := NewValue(reporter, data)
		value.chain.assertOK(t)

		for path, expected := range tests {
			actual := value.Path(path)
			actual.chain.assertOK(t)

			assert.Equal(t, expected, actual.Raw())
		}
	}

	t.Run("pick", func(t *testing.T) {
		runTests(map[string]interface{}{
			"$":         data,
			"$.A[0]":    "string",
			`$["A"][0]`: "string",
			"$.A":       []interface{}{"string", 23.3, 3.0, true, false, nil},
			"$.A[*]":    []interface{}{"string", 23.3, 3.0, true, false, nil},
			"$.A.*":     []interface{}{"string", 23.3, 3.0, true, false, nil},
			"$.A.*.a":   []interface{}{},
		})
	})

	t.Run("slice", func(t *testing.T) {
		runTests(map[string]interface{}{
			"$.A[1,4,2]":      []interface{}{23.3, false, 3.0},
			`$["B","C"]`:      []interface{}{"value", 3.14},
			`$["C","B"]`:      []interface{}{3.14, "value"},
			"$.A[1:4]":        []interface{}{23.3, 3.0, true},
			"$.A[::2]":        []interface{}{"string", 3.0, false},
			"$.A[-2:]":        []interface{}{false, nil},
			"$.A[:-1]":        []interface{}{"string", 23.3, 3.0, true, false},
			"$.A[::-1]":       []interface{}{nil, false, true, 3.0, 23.3, "string"},
			"$.F.V[4:5][0,1]": []interface{}{"string5a", "string5b"},
			"$.F.V[4:6][1]":   []interface{}{"string5b", "string6b"},
			"$.F.V[4:6][0,1]": []interface{}{"string5a", "string5b", "string6a", "string6b"},
			"$.F.V[4,5][0:2]": []interface{}{"string5a", "string5b", "string6a", "string6b"},
			"$.F.V[4:6]": []interface{}{
				[]interface{}{
					"string5a",
					"string5b",
				},
				[]interface{}{
					"string6a",
					"string6b",
				},
			},
		})
	})

	t.Run("quote", func(t *testing.T) {
		runTests(map[string]interface{}{
			`$[A][0]`:    "string",
			`$["A"][0]`:  "string",
			`$[B,C]`:     []interface{}{"value", 3.14},
			`$["B","C"]`: []interface{}{"value", 3.14},
		})
	})

	t.Run("search", func(t *testing.T) {
		runTests(map[string]interface{}{
			"$..C":       []interface{}{3.14, 3.1415, 3.141592, 3.14159265},
			`$..["C"]`:   []interface{}{3.14, 3.1415, 3.141592, 3.14159265},
			"$.D.V..C":   []interface{}{3.141592},
			"$.D.V.*.C":  []interface{}{3.141592},
			"$.D.V..*.C": []interface{}{3.141592},
			"$.D.*..C":   []interface{}{3.141592},
			"$.*.V..C":   []interface{}{3.141592},
			"$.*.D.V.C":  []interface{}{3.14159265},
			"$.*.D..C":   []interface{}{3.14159265},
			"$.*.D.V..*": []interface{}{3.14159265},
			"$..D..V..C": []interface{}{3.141592, 3.14159265},
			"$.*.*.*.C":  []interface{}{3.141592, 3.14159265},
			"$..V..C":    []interface{}{3.141592, 3.14159265},
			"$.D.V..*": []interface{}{
				"string2a",
				"string2b",
				map[string]interface{}{
					"C": 3.141592,
				},
				3.141592,
			},
			"$..A": []interface{}{
				[]interface{}{"string", 23.3, 3.0, true, false, nil},
				[]interface{}{"string3"},
			},
			"$..A..*":      []interface{}{"string", 23.3, 3.0, true, false, nil, "string3"},
			"$.A..*":       []interface{}{"string", 23.3, 3.0, true, false, nil},
			"$.A.*":        []interface{}{"string", 23.3, 3.0, true, false, nil},
			"$..A[0,1]":    []interface{}{"string", 23.3},
			"$..A[0]":      []interface{}{"string", "string3"},
			"$.*.V[0]":     []interface{}{"string2a", "string4a"},
			"$.*.V[1]":     []interface{}{"string2b", "string4b"},
			"$.*.V[0,1]":   []interface{}{"string2a", "string2b", "string4a", "string4b"},
			"$.*.V[0:2]":   []interface{}{"string2a", "string2b", "string4a", "string4b"},
			"$.*.V[2].C":   []interface{}{3.141592},
			"$..V[2].C":    []interface{}{3.141592},
			"$..V[*].C":    []interface{}{3.141592},
			"$.*.V[2].*":   []interface{}{3.141592, 3.1415926535},
			"$.*.V[2:3].*": []interface{}{3.141592, 3.1415926535},
			"$.*.V[2:4].*": []interface{}{3.141592, 3.1415926535, "hello"},
			"$..V[2,3].CC": []interface{}{3.1415926535, "hello"},
			"$..V[2:4].CC": []interface{}{3.1415926535, "hello"},
			"$..V[*].*": []interface{}{
				3.141592,
				3.1415926535,
				"hello",
				"string5a",
				"string5b",
				"string6a",
				"string6b",
			},
			"$..[0]": []interface{}{
				"string",
				"string2a",
				"string3",
				"string4a",
				"string5a",
				"string6a",
			},
			"$..ZZ": []interface{}{},
		})
	})
}

func TestValuePathIntFloat(t *testing.T) {
	reporter := newMockReporter(t)

	data := map[string]interface{}{
		"A": 123,
		"B": 123.0,
	}

	value := NewValue(reporter, data)
	value.chain.assertOK(t)

	a := value.Path(`$["A"]`)
	a.chain.assertOK(t)
	assert.Equal(t, 123.0, a.Raw())

	b := value.Path(`$["B"]`)
	b.chain.assertOK(t)
	assert.Equal(t, 123.0, b.Raw())
}

func TestValueSchema(t *testing.T) {
	reporter := newMockReporter(t)

	schema := `{
		"type": "object",
		"properties": {
			"foo": {
				"type": "string"
			},
			"bar": {
				"type": "integer"
			}
		},
		"require": ["foo", "bar"]
	}`

	data1 := map[string]interface{}{
		"foo": "a",
		"bar": 1,
	}

	data2 := map[string]interface{}{
		"foo": "a",
		"bar": "b",
	}

	NewValue(reporter, data1).Schema(schema).chain.assertOK(t)
	NewValue(reporter, data2).Schema(schema).chain.assertFailed(t)

	NewValue(reporter, data1).Schema([]byte(schema)).chain.assertOK(t)
	NewValue(reporter, data2).Schema([]byte(schema)).chain.assertFailed(t)

	var b interface{}
	json.Unmarshal([]byte(schema), &b)

	NewValue(reporter, data1).Schema(b).chain.assertOK(t)
	NewValue(reporter, data2).Schema(b).chain.assertFailed(t)

	tmp, _ := ioutil.TempFile("", "httpexpect")
	defer os.Remove(tmp.Name())

	tmp.Write([]byte(schema))
	tmp.Close()

	url := "file://" + tmp.Name()

	NewValue(reporter, data1).Schema(url).chain.assertOK(t)
	NewValue(reporter, data2).Schema(url).chain.assertFailed(t)

	NewValue(reporter, data1).Schema("file:///bad/path").chain.assertFailed(t)
	NewValue(reporter, data1).Schema("{ bad json").chain.assertFailed(t)
}
