package interpol

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

var errDummy = errors.New("dummy")

func toError(key string, w io.Writer) error {
	return errDummy
}

func toUpper(key string, w io.Writer) error {
	_, err := w.Write([]byte(strings.ToUpper(key)))
	return err
}

func toNull(key string, w io.Writer) error {
	return nil
}

func testInterpolate(template string, f Func) (string, error) {
	buffer := bytes.NewBuffer(nil)
	opts := &Options{
		Template: strings.NewReader(template),
		Output:   buffer,
		Format:   f,
	}
	i := NewWithOptions(opts)
	err := i.Interpolate()
	return buffer.String(), err
}

func TestInterpolatorEscapeOpen(t *testing.T) {
	str, err := testInterpolate("hello {{!!!{{", toUpper)
	if err != nil {
		t.Fatal(err)
	}
	if str != "hello {!!!{" {
		t.Errorf("Invalid string: %q", str)
	}
}

func TestInterpolatorEscapeClose(t *testing.T) {
	str, err := testInterpolate("hello }}!!!}}", toUpper)
	if err != nil {
		t.Fatal(err)
	}
	if str != "hello }!!!}" {
		t.Errorf("Invalid string: %q", str)
	}
}

func TestInterpolatorEscapeOpenClose(t *testing.T) {
	str, err := testInterpolate("hello }}!!!{{", toUpper)
	if err != nil {
		t.Fatal(err)
	}
	if str != "hello }!!!{" {
		t.Errorf("Invalid string: %q", str)
	}
}

func testInterpolatorClose(t *testing.T, strs []string, err error) {
	for _, str := range strs {
		strs2 := strings.Split(str, ":")
		template, output := strs2[0], strs2[1]
		output2, err2 := testInterpolate(template, toUpper)
		if err2 != err {
			t.Fatal(err2)
		}
		if output != output2 {
			t.Fatalf("Expecting %q but got %q", output, output2)
		}
	}
}

func TestInterpolatorUnexpectedClose(t *testing.T) {
	strs := []string{
		"}hello test!!!:",
		"hello test}!!!:hello test",
		"hello test!!!}:hello test!!!",
		"hello test!!!}{:hello test!!!",
	}
	testInterpolatorClose(t, strs, ErrUnexpectedClose)
}

func TestInterpolatorExpectingClose(t *testing.T) {
	strs := []string{
		"{hello test!!!:",
		"hello {test!!!:hello ",
		"hello test!!!{:hello test!!!",
	}
	testInterpolatorClose(t, strs, ErrExpectingClose)
}

type testIOFunc func(t *testing.T, template, output string)

func testInterpolateIOError(t *testing.T, f testIOFunc) {
	strs := []string{
		"こんにちは:こんにち",
		"こんにちは:こんに",
		"こんにちは:こん",
		"こんにちは:こ",
		"こんにちは:",
		"hello test!!!:",
		"hello test!!!:h",
		"hello test!!!:hello test!!",
		"{{hello test!!!:",
		"hello {{test!!!:hello ",
		"hello test!!!{{:hello test!!!",
		"}}hello test!!!:",
		"hello }}test!!!:hello ",
		"hello test!!!}}:hello test!!!",
	}
	for _, str := range strs {
		strs2 := strings.Split(str, ":")
		template, output := strs2[0], strs2[1]
		f(t, template, output)
	}
}

func testInterpolateReadError(t *testing.T, template, output string) {
	reader := newErrReader(strings.NewReader(template))
	reader.xn = len(output)
	reader.err = errDummy
	buffer := bytes.NewBuffer(nil)
	opts := &Options{
		Template: reader,
		Output:   buffer,
		Format:   toError,
	}
	i := NewWithOptions(opts)
	err := i.Interpolate()
	if err != errDummy {
		t.Fatalf("Expecting %v got %v for %q", errDummy, err, template)
	}
	if str := buffer.String(); str != output {
		t.Fatalf("Expecting %q but got %q for %q", output, str, template)
	}
}

func TestInterpolatorReadError(t *testing.T) {
	testInterpolateIOError(t, testInterpolateReadError)
}

func testInterpolateWriteError(t *testing.T, template, output string) {
	writer := newErrWriter()
	writer.xn = len(output)
	writer.err = errDummy
	opts := &Options{
		Template: strings.NewReader(template),
		Output:   writer,
		Format:   toUpper,
	}
	i := NewWithOptions(opts)
	err := i.Interpolate()
	if err != errDummy {
		t.Fatalf("Expecting %v got %v for %q", errDummy, err, template)
	}
	if str := writer.buf.String(); str != output {
		t.Fatalf("Expecting %q but got %q for %q", output, str, template)
	}
}

func TestInterpolatorWriteError(t *testing.T) {
	testInterpolateIOError(t, testInterpolateWriteError)
}

func TestInterpolatorNoRuneWriter(t *testing.T) {
	reader := strings.NewReader("{hello} {世界}!")
	template := WithTemplate(reader)
	format := WithFormat(toUpper)
	w := newErrWriter()
	w.xn = reader.Len()
	output := WithOutput(w)
	i := New(template, format, output)
	if err := i.Interpolate(); err != nil {
		t.Fatal(err)
	}
	if str := w.buf.String(); str != "HELLO 世界!" {
		t.Fatalf("Got %q", str)
	}
}

func TestWithFunc(t *testing.T) {
	str, err := WithFunc("hello {test}!!!", toUpper)
	if err != nil {
		t.Fatal(err)
	}
	if str != "hello TEST!!!" {
		t.Errorf("Invalid string: %q", str)
	}
}

func TestWithFuncFuncError(t *testing.T) {
	str, err := WithFunc("hello {test}!!!", toError)
	if len(str) != 0 || err != errDummy {
		t.Fatal(err)
	}
}

func TestWithMapKeyExists(t *testing.T) {
	m := map[string]string{
		"test": "World",
		"data": "!!!",
	}
	str, err := WithMap("hello {test}{data}", m)
	if err != nil {
		t.Fatal(err)
	}
	if str != "hello World!!!" {
		t.Errorf("Invalid string: %q", str)
	}
}

func TestWithMapKeyNotFound(t *testing.T) {
	m := map[string]string{
		"data": "World",
	}
	str, err := WithMap("hello {test}!!!", m)
	if len(str) != 0 || err != ErrKeyNotFound {
		t.Fatal(err)
	}
}

func TestWithMapNil(t *testing.T) {
	str, err := WithMap("hello {test}!!!", nil)
	if len(str) != 0 || err != ErrKeyNotFound {
		t.Fatal(err)
	}
}

func BenchmarkWithFunc0(b *testing.B) {
	benchmarkWithFuncN(b, 0)
}

func BenchmarkWithFunc1(b *testing.B) {
	benchmarkWithFuncN(b, 1)
}

func BenchmarkWithFunc1K(b *testing.B) {
	benchmarkWithFuncN(b, 1000)
}

func benchmarkWithFuncN(b *testing.B, n int) {
	b.StopTimer()
	template := strings.Repeat("?{hello}!", n)
	expected := strings.Repeat("?!", n)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		str, err := WithFunc(template, toNull)
		if err != nil {
			b.Fatal(err)
		}
		if str != expected {
			b.Fatalf("Invalid")
		}
	}
}
