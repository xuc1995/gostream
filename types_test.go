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

func TestIsFilter(t *testing.T) {
	f := func(it int) bool {
		return false
	}
	if !isFilterOf(r.TypeOf(f), r.TypeOf(0)) {
		t.Errorf("f should be a filter")
	}
}
