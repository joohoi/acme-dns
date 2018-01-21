package interpol

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestOptions(t *testing.T) {
	n := 0
	format := func(key string, w io.Writer) error {
		n++
		return nil
	}
	opts := &Options{
		Template: strings.NewReader("foo"),
		Format:   format,
		Output:   bytes.NewBuffer(nil),
	}
	opts2 := []Option{
		WithTemplate(opts.Template),
		WithFormat(opts.Format),
		WithOutput(opts.Output),
	}
	opts3 := &Options{
		Template: strings.NewReader("foo"),
		Format:   toError,
		Output:   bytes.NewBuffer(nil),
	}
	setOptions(opts2, newOptionSetter(opts3))
	if opts3.Template != opts.Template {
		t.Fatalf("Invalid template")
	}
	if opts3.Output != opts.Output {
		t.Fatalf("Invalid output")
	}
	if n != 0 || opts3.Format == nil {
		t.Fatalf("Invalid format")
	}
	if err := opts3.Format("", nil); err != nil {
		t.Fatalf("Invalid format")
	}
	if n != 1 {
		t.Fatalf("Invalid format")
	}
}
