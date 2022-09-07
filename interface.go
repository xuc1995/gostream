package gostream

import "reflect"

/****************************************************** public ********************************************************/

type Iterator interface {
	Next() bool
	Value() interface{}
}

type MapIterator interface {
	Next() bool
	Key() interface{}
	Value() interface{}
}

/***************************************************** private ********************************************************/

type iterator interface {
	Next() bool
	Value() reflect.Value
}

type mapIterator interface {
	Next() bool
	Key() reflect.Value
	Value() reflect.Value
}

type resolver interface {
	Invoke(v reflect.Value) (reflect.Value, bool)
	OutType() reflect.Type
}

type entryResolver interface {
	Invoke(k, v reflect.Value) (reflect.Value, reflect.Value, bool)
	OutKeyType() reflect.Type
	OutValueType() reflect.Type
}
