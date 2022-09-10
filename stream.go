package gostream

import (
	"errors"
	"fmt"
	r "reflect"
)

var errNilParameter = errors.New("nil parameter is ill illegal")

/************************************************* sequence stream ****************************************************/

type ObjectStream struct {
	inType       r.Type
	iter         iterator
	resolveChain []resolver
	err          error
}

// S is short for SliceStream
func S(sequence interface{}) *ObjectStream {
	return SliceStream(sequence)
}

func SliceStream(sequence interface{}) *ObjectStream {
	iter, err := iter(sequence)
	if err != nil {
		return &ObjectStream{err: err}
	}
	value := r.ValueOf(sequence)
	if !isSequence(value) {
		return &ObjectStream{err: fmt.Errorf("parameter is not sequence but: %T", sequence)}
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
		return &ObjectStream{err: errNilParameter}
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

type objectToEntryIter struct {
	innerIter iterator
	mapper    r.Value

	// It is important to ensure that mapper func is invoked only once during on Next() call
	mapperCalled bool
	k            r.Value
	v            r.Value
}

func (i *objectToEntryIter) Next() bool {
	hasNext := i.innerIter.Next()
	i.mapperCalled = false
	return hasNext
}

func (i objectToEntryIter) Key() r.Value {
	if i.mapperCalled {
		return i.k
	}
	callResult := i.mapper.Call(p{i.innerIter.Value()})
	i.mapperCalled = true
	i.k = callResult[0]
	i.v = callResult[1]
	return i.k
}

func (i objectToEntryIter) Value() r.Value {
	if i.mapperCalled {
		return i.v
	}
	callResult := i.mapper.Call(p{i.innerIter.Value()})
	i.mapperCalled = true
	i.k = callResult[0]
	i.v = callResult[1]
	return i.v
}

func (s *ObjectStream) toEntryIterator(mapper interface{}) mapIterator {
	return &objectToEntryIter{
		innerIter: &objectStreamIterator{stream: s},
		mapper:    r.ValueOf(mapper),
	}
}

/************************************************ sequence stream exported ********************************************/

func (s *ObjectStream) ToEntryStream(mapper interface{}) *MapEntryStream {
	if s.err != nil {
		return &MapEntryStream{err: s.err}
	}
	mapperType := r.TypeOf(mapper)
	inKeyType := mapperType.Out(0)
	inValueType := mapperType.Out(1)
	if !isObjectToEntryAdaptorOf(mapperType, s.outType()) {
		return &MapEntryStream{err: fmt.Errorf("paramter mapper error: %T", mapper)}
	}
	return &MapEntryStream{
		iter:        s.toEntryIterator(mapper),
		inKeyType:   inKeyType,
		inValueType: inValueType,
	}
}

func (s *ObjectStream) Filter(filter interface{}) *ObjectStream {
	if s.err != nil {
		return s
	}
	fv := r.ValueOf(filter)
	oType := s.outType()
	if !isFilterOf(fv.Type(), s.outType()) || fv.Type().In(0).Kind() != oType.Kind() {
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
	if !isMapperOf(mv.Type(), inType) {
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
	if value.Kind() != r.Ptr {
		return fmt.Errorf("parameter of CollectAt should be pointer, but is: %T", at)
	}
	target := value.Elem()
	if target.Kind() != r.Slice {
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

// ES is short for EntryStream
func ES(anyMap interface{}) *MapEntryStream {
	return EntryStream(anyMap)
}

func EntryStream(anyMap interface{}) *MapEntryStream {
	iter, err := iterMap(anyMap)
	if err != nil {
		return &MapEntryStream{err: err}
	}
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
	if !isEntryFilterOf(fv.Type(), keyType, valueType) {
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
	if pointer.Kind() != r.Ptr {
		return fmt.Errorf("parameter of CollectAt should be pointer, but is: %T", at)
	}
	value := pointer.Elem()
	if !isMapOf(value.Type(), ms.outKeyType(), ms.outValueType()) {
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
