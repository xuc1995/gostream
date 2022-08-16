package gostream

import r "reflect"

/************************************************* slice stream *******************************************************/

type ObjectStream struct {
	inType       r.Type
	iter         Iterator
	resolveChain []Resolver
	err          error
}

func SliceStream(anySlice SliceOfT) *ObjectStream {
	iter, err := Iter(anySlice)
	if err != nil {
		return &ObjectStream{err: err}
	}
	// TODO do type check
	sliceType := r.TypeOf(anySlice)
	return &ObjectStream{iter: iter, inType: sliceType.Elem()}
}

func Stream(iterator Iterator, emptyContent interface{}) *ObjectStream {
	// TODO do type check
	return &ObjectStream{iter: iterator, inType: r.TypeOf(emptyContent)}
}

func (s *ObjectStream) outType() r.Type {
	chainLen := len(s.resolveChain)
	if chainLen == 0 {
		return s.inType
	}
	return s.resolveChain[chainLen-1].OutType()
}

/************************************************ slice stream exported ***********************************************/

type objectStreamIterator struct {
	stream    *ObjectStream
	nextValue r.Value
}

func (i *objectStreamIterator) Next() bool {
	// find if have nextValue and put it at i.nextValue
	iter := i.stream.iter
	for iter.Next() {
		next := iter.Value()
		value := r.ValueOf(next)
		valid := true
		for _, fun := range i.stream.resolveChain {
			result := fun.Invoke(value)
			if result.Ok() {
				value = result.Result()
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

func (i *objectStreamIterator) Value() interface{} {
	return i.nextValue.Interface()
}

// *ObjectStream self is iterable.
func (s *ObjectStream) Iter() Iterator {
	return &objectStreamIterator{stream: s}
}

type objectSteamAsKeyIter struct {
	innerIter Iterator
	mapper    r.Value
}

func (i *objectSteamAsKeyIter) Next() bool {
	return i.innerIter.Next()
}

func (i objectSteamAsKeyIter) Entry() Entry {
	next := i.innerIter.Value()
	return &mapEntry{
		key:   next,
		value: i.mapper.Call([]r.Value{r.ValueOf(next)})[0].Interface(),
	}
}

func (s *ObjectStream) IterAsMapKey(mapper Mapper) MapIterator {
	// TODO do type check
	return &objectSteamAsKeyIter{
		innerIter: s.Iter(),
		mapper:    r.ValueOf(mapper),
	}
}

// AsMapKey produce a MapEntryStream which using element in this object stream as map's key, and mapper(element)
// as map's value from this ObjectStream
func (s *ObjectStream) AsMapKey(mapper Mapper) *MapEntryStream {
	if s.err != nil {
		return &MapEntryStream{err: s.err}
	}
	// TODO do type check
	mapInType := s.outType()
	mapOutType := r.TypeOf(mapper).Out(0)
	return &MapEntryStream{
		iter:        s.IterAsMapKey(mapper),
		inKeyType:   mapInType,
		inValueType: mapOutType,
	}
}

// Resolve Apply a function on every element of a stream.
/**/
// Typically is not usefully, because "Map" and "Filter" cover almost
// all conventional conditions.
// It is somehow UNSAFE, because the real rtype of invoke().result have not been checked which considering
// be the same as what Resolver's OutType() pointer out.
func (s *ObjectStream) Resolve(resolver Resolver) *ObjectStream {
	if s.err != nil {
		return s
	}
	// TODO do type check
	s.resolveChain = append(s.resolveChain, resolver)
	return s

}

func (s *ObjectStream) Filter(filter Filter) *ObjectStream {
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

func (s *ObjectStream) Map(mapper Mapper) *ObjectStream {
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

func (s *ObjectStream) Collect() (SliceOfU, error) {
	if s.err != nil {
		return nil, s.err
	}
	outType := s.outType()
	resValue := r.MakeSlice(r.SliceOf(outType), 0, 0)
	return s.collect(resValue).interfaceOrErr()
}

func (s *ObjectStream) CollectAt(u SliceOfU) error {
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
		next := s.iter.Value()
		value := r.ValueOf(next)
		valid := true
		for _, fun := range s.resolveChain {
			if result := fun.Invoke(value); result.Ok() {
				value = result.Result()
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

/************************************************ resolve result ******************************************************/

type resolveResult struct {
	result r.Value
	valid  bool
}

func (i *resolveResult) Result() r.Value {
	return i.result
}

func (i *resolveResult) Ok() bool {
	return i.valid
}

/********************************************* slice stream resolver **************************************************/

type mapperFun struct {
	f       r.Value
	inType  r.Type
	outType r.Type
}

func (m *mapperFun) OutType() r.Type {
	return m.outType
}

func (m *mapperFun) Invoke(v r.Value) ResolveResult {
	return &resolveResult{result: m.f.Call([]r.Value{v})[0], valid: true}
}

type filterFun struct {
	f           r.Value
	contentType r.Type
}

func (f *filterFun) Invoke(v r.Value) ResolveResult {
	return &resolveResult{result: v, valid: f.f.Call([]r.Value{v})[0].Bool()}
}

func (f *filterFun) OutType() r.Type {
	return f.contentType
}

/********************************************* map entry stream *******************************************************/

type MapEntryStream struct {
	err               error
	inKeyType         r.Type
	inValueType       r.Type
	iter              MapIterator
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

func EntryStream(anyMap MapOfKV) *MapEntryStream {
	iter, err := IterMap(anyMap)
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

func (ms *MapEntryStream) Filter(filter Filter) *MapEntryStream {
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

func (ms *MapEntryStream) Collect() (MapOfUW, error) {
	if ms.err != nil {
		return nil, ms.err
	}
	// TODO do type check
	outKeyType := ms.outKeyType()
	outValueType := ms.outValueType()
	resMap := r.MakeMap(r.MapOf(outKeyType, outValueType))

	return ms.collect(resMap).interfaceOrErr()
}

func (ms *MapEntryStream) CollectAt(uw MapOfUW) error {
	targetMap := r.ValueOf(uw).Elem()
	// TODO do type check
	return ms.collect(targetMap).writeBack()
}

// #no-type-check; #no-error-check
func (ms *MapEntryStream) collect(target r.Value) *collectResult {
	for ms.iter.Next() {
		next := ms.iter.Entry()
		valid := true
		var entryValue = &entry{k: r.ValueOf(next.Key()), v: r.ValueOf(next.Value())}
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
