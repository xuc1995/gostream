package gostream

import r "reflect"

/************************************************* sequence stream ****************************************************/

type ObjectStream struct {
	inType       r.Type
	iter         iterator
	resolveChain []resolver
	err          error
}

func SliceStream(anySlice interface{}) *ObjectStream {
	iter, err := iter(anySlice)
	if err != nil {
		return &ObjectStream{err: err}
	}
	// TODO do type check
	sliceType := r.TypeOf(anySlice)
	return &ObjectStream{iter: iter, inType: sliceType.Elem()}
}

func Stream(iterator Iterator, Type interface{}) *ObjectStream {
	// TODO do type check
	return &ObjectStream{iter: iterAdaptor{iterator}, inType: r.TypeOf(Type)}
}

func (s *ObjectStream) outType() r.Type {
	chainLen := len(s.resolveChain)
	if chainLen == 0 {
		return s.inType
	}
	return s.resolveChain[chainLen-1].OutType()
}

/************************************************ sequence stream exported ********************************************/

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
	next := i.innerIter.Value()
	return next
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
	// TODO do type check
	s.resolveChain = append(s.resolveChain, &filterFun{
		f:           r.ValueOf(filter),
		contentType: s.outType(),
	})
	return s
}

func (s *ObjectStream) Map(mapper interface{}) *ObjectStream {
	if s.err != nil {
		return s
	}
	// TODO do type check
	s.resolveChain = append(s.resolveChain, &mapperFun{
		f:       r.ValueOf(mapper),
		inType:  s.outType(),
		outType: r.TypeOf(mapper).Out(0),
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

func (s *ObjectStream) CollectAt(u interface{}) error {
	if s.err != nil {
		return s.err
	}
	// TODO do type check
	slice := r.ValueOf(u).Elem()
	return s.collect(slice).writeBack()
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
	entryResolveChain []*entryResolver
}

type entryResolver struct {
	fn           r.Value // func(entry)
	outKeyType   r.Type
	outValueType r.Type
}

func (er *entryResolver) invoke(e *entry) *entryResolveResult {
	ok := er.fn.Call([]r.Value{e.k, e.v})[0].Bool()
	if ok {
		return &entryResolveResult{
			ok:     true,
			result: e,
		}
	}
	return &entryResolveResult{ok: false}
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
		entryResolveChain: make([]*entryResolver, 0, 0),
	}
}

func (ms *MapEntryStream) outKeyType() r.Type {
	if len(ms.entryResolveChain) > 0 {
		return ms.entryResolveChain[len(ms.entryResolveChain)-1].outKeyType
	}
	return ms.inKeyType
}

func (ms *MapEntryStream) outValueType() r.Type {
	if len(ms.entryResolveChain) > 0 {
		return ms.entryResolveChain[len(ms.entryResolveChain)-1].outValueType
	}
	return ms.inValueType
}

/********************************************* entry stream exported **************************************************/

func (ms *MapEntryStream) Filter(filter interface{}) *MapEntryStream {
	if ms.err != nil {
		return ms
	}
	// TODO do type check
	ms.entryResolveChain = append(ms.entryResolveChain, &entryResolver{
		fn:           r.ValueOf(filter),
		outKeyType:   ms.outKeyType(),
		outValueType: ms.outValueType(),
	})
	return ms
}

func (ms *MapEntryStream) Collect() (interface{}, error) {
	if ms.err != nil {
		return nil, ms.err
	}
	// TODO do type check
	outKeyType := ms.outKeyType()
	outValueType := ms.outValueType()
	resMap := r.MakeMap(r.MapOf(outKeyType, outValueType))

	return ms.collect(resMap).interfaceOrErr()
}

func (ms *MapEntryStream) CollectAt(uw interface{}) error {
	targetMap := r.ValueOf(uw).Elem()
	// TODO do type check
	return ms.collect(targetMap).writeBack()
}

// #no-type-check; #no-error-check
func (ms *MapEntryStream) collect(target r.Value) *collectResult {
	for ms.iter.Next() {
		valid := true
		var entryValue = &entry{k: ms.iter.Key(), v: ms.iter.Value()}
		for _, fun := range ms.entryResolveChain {
			result := fun.invoke(entryValue)
			if result.ok {
				entryValue = result.result
				continue
			}
			valid = false
			break
		}
		if valid {
			target.SetMapIndex(entryValue.k, entryValue.v)
		}
	}
	return &collectResult{origin: target, v: target, err: nil}
}

/********************************************* entry resolve result ***************************************************/

type entryResolveResult struct {
	result *entry
	ok     bool
}

type entry struct {
	k r.Value
	v r.Value
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
