package gostream

import r "reflect"

type p []r.Value

func isSequence(v r.Value) bool {
	kind := v.Kind()
	return kind == r.Slice || kind == r.Array
}

func isSlice(v r.Value) bool {
	return v.IsValid() && v.Type().Kind() == r.Slice
}

func isMap(v r.Value) bool {
	if v.IsValid() {
		return v.Type().Kind() == r.Map
	}
	return false
}

func isMapOf(m r.Value, kType, vType r.Type) bool {
	return isMap(m) &&
		m.Type().Key().Kind() == kType.Kind() &&
		m.Type().Elem().Kind() == vType.Kind()
}

func isFilterOf(f r.Value, itemType r.Type) bool {
	if !f.IsValid() {
		return false
	}
	typ := f.Type()
	return typ.Kind() == r.Func &&
		typ.NumIn() == 1 &&
		typ.NumOut() == 1 &&
		typ.In(0).Kind() == itemType.Kind() &&
		typ.Out(0).Kind() == r.Bool
}

func isMapperOf(f r.Value, inType r.Kind) bool {
	if !f.IsValid() {
		return false
	}
	typ := f.Type()
	return typ.Kind() == r.Func &&
		typ.NumIn() == 1 &&
		typ.NumOut() == 1 &&
		typ.In(0).Kind() == inType
}

func isPointer(p r.Value) bool {
	if !p.IsValid() {
		return false
	}
	return p.Kind() == r.Ptr
}

func isEntryFilterOf(f r.Value, kType, vType r.Type) bool {
	if !f.IsValid() {
		return false
	}
	typ := f.Type()
	return typ.Kind() == r.Func &&
		typ.NumIn() == 2 &&
		typ.NumOut() == 1 &&
		typ.Out(0).Kind() == r.Bool &&
		typ.In(0).Kind() == kType.Kind() &&
		typ.In(1).Kind() == vType.Kind()
}
