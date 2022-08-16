package gostream

import r "reflect"

type iterator struct {
	slice        r.Value
	currentIndex int
}

func (i *iterator) Next() bool {
	return i.currentIndex < i.slice.Len()
}

func (i *iterator) Value() interface{} {
	next := i.slice.Index(i.currentIndex)
	i.currentIndex++
	return next.Interface()
}

type mapEntry struct {
	key   interface{}
	value interface{}
}

func (e *mapEntry) Key() interface{} {
	return e.key
}

func (e *mapEntry) Value() interface{} {
	return e.value
}

type mapIterator struct {
	mapIter *r.MapIter
}

func (i *mapIterator) Next() bool {
	return i.mapIter.Next()
}

func (i *mapIterator) Entry() Entry {
	return &mapEntry{
		key:   i.mapIter.Key().Interface(),
		value: i.mapIter.Value().Interface(),
	}
}

func Iter(anySlice SliceOfT) (Iterator, error) {
	// TODO do type check
	return &iterator{
		slice: r.ValueOf(anySlice),
	}, nil
}

func IterMap(anyMap MapOfKV) (MapIterator, error) {
	// TODO do type check
	mapValue := r.ValueOf(anyMap)
	return &mapIterator{
		mapIter: mapValue.MapRange(),
	}, nil
}
