package gostream

import r "reflect"

// Do type check in this file.

func isSequence(v r.Value) bool {
	if v.IsValid() {
		kind := v.Type().Kind()
		return kind == r.Slice || kind == r.Array
	}
	return false
}

func isMap(v r.Value) bool {
	if v.IsValid() {
		return v.Type().Kind() == r.Map
	}
	return false
}
