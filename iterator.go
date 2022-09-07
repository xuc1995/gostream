package gostream

import (
	"fmt"
	r "reflect"
)

type iterAdaptor struct{ Iterator }

func (i iterAdaptor) Value() r.Value {
	return r.ValueOf(i.Iterator.Value())
}

type sequenceIterator struct {
	sequence     r.Value
	currentIndex int
}

func (i *sequenceIterator) Next() bool {
	return i.currentIndex < i.sequence.Len()
}

func (i *sequenceIterator) Value() r.Value {
	next := i.sequence.Index(i.currentIndex)
	i.currentIndex++
	return next
}

func iter(anySlice interface{}) (iterator, error) {
	v := r.ValueOf(anySlice)
	if !isSequence(v) {
		return nil, fmt.Errorf("parameter is not of type sequence or array, whitch is: %T", anySlice)
	}
	return &sequenceIterator{
		sequence: v,
	}, nil
}

func iterMap(anyMap interface{}) (mapIterator, error) {
	mapValue := r.ValueOf(anyMap)
	if mapValue.Kind() != r.Map {
		return nil, fmt.Errorf("parameter is not of type map, which is: %T", anyMap)
	}
	return mapValue.MapRange(), nil
}
