package gostream

import (
	"errors"
	"fmt"
	r "reflect"
)

var errParameterNotASequence = errors.New("parameter is not a sequence")

var errParameterTypeNil = errors.New("parameter itemType is nil")

/************************************************* sequence stream ****************************************************/

type ObjectStream struct {
	inType       r.Type
	iter         iterator
	resolveChain []resolver
	err          error
}

func SliceStream(sequence interface{}) *ObjectStream {
	iter, err := iter(sequence)
	if err != nil {
		return &ObjectStream{err: err}
	}
	value := r.ValueOf(sequence)
	if !isSequence(value) {
		return &ObjectStream{err: errParameterNotASequence}
	}
	typ := r.TypeOf(sequence)
	return &ObjectStream{iter: iter, inType: typ.Elem()}
}

// Stream make an ObjectStream from an iterator.
//
// Before iterator invoked, out type of iterator could not be known, so make sure itemType is actually correct,
// or it would cause runtime panic
func Stream(iterator Iterator, itemType interface{}) *ObjectStream {
	if !r.ValueOf(itemType).IsValid() {
		return &ObjectStream{err: errParameterTypeNil}
	}
	return &ObjectStream{iter: iterAdaptor{iterator}, inType: r.TypeOf(itemType)}
}

func (s *ObjectStream) outType() r.Type {
	chainLen := len(s.resolveChain)
	if chainLen == 0 {
		return s.inType
	}
	return s.resolveChain[chainLen-1].OutType()
}

type objectStreamIterator struct {
	stream    *ObjectStream
	nextValue r.Value
}

func (i *objectStreamIterator) Next() bool {
	// find if have nextValue and put it at i.nextValue
	iter := i.stream.iter
	for iter.Next() {
		value := iter.Value()
		valid := true
		for _, fun := range i.stream.resolveChain {
			if result, ok := fun.Invoke(value); ok {
				value = result
				continue
			}
			valid = false
			break
		}
		if valid { // find valid nextValue
			i.nextValue = value
			return true
		}
	}
	return false
}

func (i *objectStreamIterator) Value() r.Value {
	return i.nextValue
}

type objectSteamAsKeyIter struct {
	innerIter iterator
	mapper    r.Value
}

func (i *objectSteamAsKeyIter) Next() bool {
	return i.innerIter.Next()
}

func (i objectSteamAsKeyIter) Key() r.Value {
	return i.innerIter.Value()
}

func (i objectSteamAsKeyIter) Value() r.Value {
	return i.mapper.Call([]r.Value{i.innerIter.Value()})[0]
}

func (s *ObjectStream) iterAsMapKey(mapper interface{}) mapIterator {
	// TODO do type check
	return &objectSteamAsKeyIter{
		innerIter: &objectStreamIterator{stream: s},
		mapper:    r.ValueOf(mapper),
	}
}

/************************************************ sequence stream exported ********************************************/

// AsMapKey produce a MapEntryStream which using element in this object stream as map's key, and mapper(element)
// as map's value from this ObjectStream
func (s *ObjectStream) AsMapKey(mapper interface{}) *MapEntryStream {
	if s.err != nil {
		return &MapEntryStream{err: s.err}
	}
	// TODO do type check
	mapInType := s.outType()
	mapOutType := r.TypeOf(mapper).Out(0)
	return &MapEntryStream{
		iter:        s.iterAsMapKey(mapper),
		inKeyType:   mapInType,
		inValueType: mapOutType,
	}
}

func (s *ObjectStream) Filter(filter interface{}) *ObjectStream {
	if s.err != nil {
		return s
	}
	fv := r.ValueOf(filter)
	oType := s.outType()
	if !isFilterOf(fv, s.outType()) || fv.Type().In(0).Kind() != oType.Kind() {
		s.err = fmt.Errorf("filter type error, which is: %T", filter)
		return s
	}
	s.resolveChain = append(s.resolveChain, &filterFun{
		f:           fv,
		contentType: oType,
	})
	return s
}

func (s *ObjectStream) Map(mapper interface{}) *ObjectStream {
	if s.err != nil {
		return s
	}
	mv := r.ValueOf(mapper)
	inType := s.outType()
	if !isMapperOf(mv, inType.Kind()) {
		s.err = fmt.Errorf("mapper type error, which is: %T", mapper)
		return s
	}
	outType := r.TypeOf(mapper).Out(0)
	s.resolveChain = append(s.resolveChain, &mapperFun{
		f:       mv,
		inType:  inType,
		outType: outType,
	})
	return s
}

func (s *ObjectStream) Collect() (interface{}, error) {
	if s.err != nil {
		return nil, s.err
	}
	outType := s.outType()
	resValue := r.MakeSlice(r.SliceOf(outType), 0, 0)
	return s.collect(resValue).interfaceOrErr()
}

func (s *ObjectStream) CollectAt(at interface{}) error {
	if s.err != nil {
		return s.err
	}
	value := r.ValueOf(at)
	if !isPointer(value) {
		return fmt.Errorf("parameter of CollectAt should be pointer, but is: %T", at)
	}
	target := value.Elem()
	if !isSlice(target) {
		return fmt.Errorf("parameter of CollectAt should be pointer of slice, but is: %s", target.Kind().String())
	}
	elemType := target.Type().Elem()
	if elemType.Kind() != s.outType().Kind() {
		return fmt.Errorf("stream's element out type is: %s, while target elemet type is: %s", s.outType().String(), elemType.String())
	}
	return s.collect(target).writeBack()
}

// #no-type-check; #no-error-check
func (s *ObjectStream) collect(target r.Value) *collectResult {
	origin := target
	for s.iter.Next() {
		value := s.iter.Value()
		valid := true
		for _, fun := range s.resolveChain {
			if result, ok := fun.Invoke(value); ok {
				value = result
			} else {
				valid = false
				break
			}
		}
		if valid {
			target = r.Append(target, value)
		}
	}
	return &collectResult{origin, target, nil}
}

/********************************************* sequence stream resolver ***********************************************/

type mapperFun struct {
	f       r.Value
	inType  r.Type
	outType r.Type
}

func (m *mapperFun) OutType() r.Type {
	return m.outType
}

func (m *mapperFun) Invoke(v r.Value) (r.Value, bool) {
	return m.f.Call([]r.Value{v})[0], true
}

type filterFun struct {
	f           r.Value
	contentType r.Type
}

func (f *filterFun) Invoke(v r.Value) (r.Value, bool) {
	return v, f.f.Call([]r.Value{v})[0].Bool()
}

func (f *filterFun) OutType() r.Type {
	return f.contentType
}

/********************************************* map entry stream *******************************************************/

type MapEntryStream struct {
	err               error
	inKeyType         r.Type
	inValueType       r.Type
	iter              mapIterator
	entryResolveChain []entryResolver
}

func EntryStream(anyMap interface{}) *MapEntryStream {
	iter, err := iterMap(anyMap)
	if err != nil {
		return &MapEntryStream{err: err}
	}
	// TODO do type check
	mapType := r.TypeOf(anyMap)
	keyType := mapType.Key()
	valueType := mapType.Elem()
	return &MapEntryStream{
		inKeyType:         keyType,
		inValueType:       valueType,
		iter:              iter,
		entryResolveChain: make([]entryResolver, 0, 0),
	}
}

func (ms *MapEntryStream) outKeyType() r.Type {
	if len(ms.entryResolveChain) > 0 {
		return ms.entryResolveChain[len(ms.entryResolveChain)-1].OutKeyType()
	}
	return ms.inKeyType
}

func (ms *MapEntryStream) outValueType() r.Type {
	if len(ms.entryResolveChain) > 0 {
		return ms.entryResolveChain[len(ms.entryResolveChain)-1].OutValueType()
	}
	return ms.inValueType
}

/********************************************* entry stream exported **************************************************/

type entryFilter struct {
	fn           r.Value // func(entry)
	outKeyType   r.Type
	outValueType r.Type
}

func (er *entryFilter) Invoke(k, v r.Value) (r.Value, r.Value, bool) {
	ok := er.fn.Call(p{k, v})[0].Bool()
	return k, v, ok
}

func (er *entryFilter) OutKeyType() r.Type {
	return er.outKeyType
}

func (er *entryFilter) OutValueType() r.Type {
	return er.outValueType
}

func (ms *MapEntryStream) Filter(filter interface{}) *MapEntryStream {
	if ms.err != nil {
		return ms
	}
	fv := r.ValueOf(filter)
	keyType := ms.outKeyType()
	valueType := ms.outValueType()
	if !isEntryFilterOf(fv, keyType, valueType) {
		ms.err = fmt.Errorf("filter type error: %T", filter)
		return ms
	}
	ms.entryResolveChain = append(ms.entryResolveChain, &entryFilter{
		fn:           fv,
		outKeyType:   keyType,
		outValueType: valueType,
	})
	return ms
}

func (ms *MapEntryStream) Collect() (interface{}, error) {
	if ms.err != nil {
		return nil, ms.err
	}
	outKeyType := ms.outKeyType()
	outValueType := ms.outValueType()
	resMap := r.MakeMap(r.MapOf(outKeyType, outValueType))
	return ms.collect(resMap).interfaceOrErr()
}

func (ms *MapEntryStream) CollectAt(at interface{}) error {
	if ms.err != nil {
		return ms.err
	}
	pointer := r.ValueOf(at)
	if !isPointer(pointer) {
		return fmt.Errorf("parameter of CollectAt should be pointer, but is: %T", at)
	}
	value := pointer.Elem()
	if !isMapOf(value, ms.outKeyType(), ms.outValueType()) {
		return fmt.Errorf("parameter type error: %T", at)
	}
	return ms.collect(value).writeBack()
}

// #no-type-check; #no-error-check
func (ms *MapEntryStream) collect(target r.Value) *collectResult {
	for ms.iter.Next() {
		valid := true
		k, v, ok := ms.iter.Key(), ms.iter.Value(), false
		for _, fun := range ms.entryResolveChain {
			k, v, ok = fun.Invoke(k, v)
			if ok {
				continue
			}
			valid = false
			break
		}
		if valid {
			target.SetMapIndex(k, v)
		}
	}
	return &collectResult{origin: target, v: target, err: nil}
}

/************************************************** stream common *****************************************************/

type collectResult struct {
	origin r.Value
	v      r.Value
	err    error
}

func (cr *collectResult) interfaceOrErr() (interface{}, error) {
	if cr.err != nil {
		return nil, cr.err
	}
	return cr.v.Interface(), nil
}

func (cr *collectResult) writeBack() error {
	if cr.err != nil {
		return cr.err
	}
	cr.origin.Set(cr.v)
	return nil
}
