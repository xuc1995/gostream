package gostream

import r "reflect"

type SliceOfT interface{}

type SliceOfU interface{}

type MapOfKV interface{}

type MapOfUW interface{}

type ResolveFun interface{}

type Mapper ResolveFun

type Filter ResolveFun

type IIterator interface {
	HasNext() bool
	Next() interface{}
}

type IMapEntry interface {
	Key() interface{}
	Value() interface{}
}

type IMapIterator interface {
	HasNext() bool
	Next() IMapEntry
}

type IResolveResult interface {
	Result() r.Value
	Ok() bool
}

type IResolver interface {
	Invoke(v r.Value) IResolveResult
	OutType() r.Type
}

type IEntry interface {
	Key() r.Value
	Value() r.Value
}

type IEntryResolveResult interface {
	Result() IEntry
	Ok() bool
}

type IEntryResolver interface {
	Invoke(e IEntry) IEntryResolveResult
	OutKeyType() r.Type
	OutValueType() r.Type
}
