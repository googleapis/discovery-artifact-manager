package clone

import (
	"reflect"
	"testing"
)

type inner struct {
	Name   string
	Values []string
}

type outer struct {
	A int
	B float32
	C string
	D []string
	E map[string]*inner
}

func TestClone(t *testing.T) {
	data := &outer{
		A: 1,
		B: 2.3,
		C: "zap",
		D: []string{"yow", "xoom"},
		E: map[string]*inner{
			"wow":    {"veep", []string{"umph", "toot"}},
			"sheesh": {"rawr", []string{"qqq", "poof"}},
		},
	}

	var cloned *outer

	if err := Clone(data, &cloned); err != nil {
		t.Errorf("unexpected clone error: %s", err)
	}
	if !reflect.DeepEqual(data, cloned) {
		t.Errorf("data not cloned correctly:\n   data:   %#v\n   cloned: %#v\n\n", data, cloned)
	}
}
