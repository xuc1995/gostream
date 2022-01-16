package gostream

import r "reflect"

type Iterator struct {
	slice        r.Value
	currentIndex int
}

func (i *Iterator) HasNext() bool {
	return i.currentIndex < i.slice.Len()
}

func (i *Iterator) Next() interface{} {
	next := i.slice.Index(i.currentIndex)
	i.currentIndex++
	return next.Interface()
}

type MapEntry struct {
	key   interface{}
	value interface{}
}

func (e *MapEntry) Key() interface{} {
	return e.key
}

func (e *MapEntry) Value() interface{} {
	return e.value
}

type MapIterator struct {
	mapIter *r.MapIter
}

func (i *MapIterator) HasNext() bool {
	return i.mapIter.Next()
}

func (i *MapIterator) Next() IMapEntry {
	return &MapEntry{
		key:   i.mapIter.Key().Interface(),
		value: i.mapIter.Value().Interface(),
	}
}

func Iter(anySlice SliceOfT) (IIterator, error) {
	// TODO do type check
	return &Iterator{
		slice: r.ValueOf(anySlice),
	}, nil
}

func IterMap(anyMap MapOfKV) (IMapIterator, error) {
	// TODO do type check
	mapValue := r.ValueOf(anyMap)
	return &MapIterator{
		mapIter: mapValue.MapRange(),
	}, nil
}
