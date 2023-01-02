package acmedns

import "testing"

func TestAllowedFrom(t *testing.T) {
	testslice := NewACMETxt()
	testslice.AllowFrom = []string{"192.168.1.0/24", "2001:db8::/32"}
	for _, test := range []struct {
		input    string
		expected bool
	}{
		{"192.168.1.42", true},
		{"192.168.2.42", false},
		{"2001:db8:aaaa::", true},
		{"2001:db9:aaaa::", false},
	} {
		if testslice.AllowedFrom(test.input) != test.expected {
			t.Errorf("Was expecting AllowedFrom to return %t for %s but got %t instead.", test.expected, test.input, !test.expected)
		}
	}
}

func TestAllowedFromList(t *testing.T) {
	testslice := ACMETxt{AllowFrom: []string{"192.168.1.0/24", "2001:db8::/32"}}
	if testslice.AllowedFromList([]string{"192.168.2.2", "1.1.1.1"}) != false {
		t.Errorf("Was expecting AllowedFromList to return false")
	}
	if testslice.AllowedFromList([]string{"192.168.1.2", "1.1.1.1"}) != true {
		t.Errorf("Was expecting AllowedFromList to return true")
	}
	allowfromall := ACMETxt{AllowFrom: []string{}}
	if allowfromall.AllowedFromList([]string{"192.168.1.2", "1.1.1.1"}) != true {
		t.Errorf("Expected non-restricted AlloFrom to be allowed")
	}
	if allowfromall.AllowedFromList([]string{}) != true {
		t.Errorf("Expected non-restricted AlloFrom to be allowed for empty list")
	}
}
