package gostream

import r "reflect"

type p []r.Value

func isSequence(v r.Value) bool {
	kind := v.Kind()
	return kind == r.Slice || kind == r.Array
}

func isMapOf(typ, kType, vType r.Type) bool {
	return typ.Kind() == r.Map &&
		typ.Key().Kind() == kType.Kind() &&
		typ.Elem().Kind() == vType.Kind()
}

func isFilterOf(typ, itemType r.Type) bool {
	return typ.Kind() == r.Func &&
		typ.NumIn() == 1 &&
		typ.NumOut() == 1 &&
		typ.In(0).Kind() == itemType.Kind() &&
		typ.Out(0).Kind() == r.Bool
}

func isMapperOf(typ, inType r.Type) bool {
	return typ.Kind() == r.Func &&
		typ.NumIn() == 1 &&
		typ.NumOut() == 1 &&
		typ.In(0).Kind() == inType.Kind()
}

func isEntryFilterOf(typ, kType, vType r.Type) bool {
	return typ.Kind() == r.Func &&
		typ.NumIn() == 2 &&
		typ.NumOut() == 1 &&
		typ.Out(0).Kind() == r.Bool &&
		typ.In(0).Kind() == kType.Kind() &&
		typ.In(1).Kind() == vType.Kind()
}

func isObjectToEntryAdaptorOf(adaptorType, elemType r.Type) bool {
	return adaptorType.Kind() == r.Func &&
		adaptorType.NumIn() == 1 &&
		adaptorType.NumOut() == 2 &&
		adaptorType.In(0).Kind() == elemType.Kind()
}
