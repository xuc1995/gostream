package gostream

import (
	r "reflect"
	"testing"
)

func TestIsSequence(t *testing.T) {
	var slice []int
	if !isSequence(r.ValueOf(slice)) {
		t.Errorf("sequence should be a sequnce")
	}
	var array = [0]int{}
	if !isSequence(r.ValueOf(array)) {
		t.Errorf("array should be a sequence")
	}
	var noType interface{}
	if isSequence(r.ValueOf(noType)) {
		t.Errorf("nil interface{} should not be a sequence")
	}
}
