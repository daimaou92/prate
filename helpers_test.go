package prate

import (
	"fmt"
	"testing"
)

func TestWrapErr(t *testing.T) {
	p := "test error"
	o := fmt.Sprintf("github.com/daimaou92/gate.TestWrapErr -> %s", p)
	v := wrapErr(fmt.Errorf(p)).Error()
	if o != v {
		t.Fatalf("wanted: %s. got: %s", o, v)
	}
}
