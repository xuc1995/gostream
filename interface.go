package gostream

import r "reflect"

type Iterator interface {
	Next() bool
	Value() interface{}
}

type Entry interface {
	Key() interface{}
	Value() interface{}
}

type MapIterator interface {
	Next() bool
	Entry() Entry
}

type ResolveResult interface {
	Result() r.Value
	Ok() bool
}

type Resolver interface {
	Invoke(v r.Value) ResolveResult
	OutType() r.Type
}
