package gostream

import r "reflect"

/************************************************* slice stream *******************************************************/

type ObjectStream struct {
	inType       r.Type
	iter         IIterator
	resolveChain []IResolver
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

func Stream(iterator IIterator, emptyContent interface{}) *ObjectStream {
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

// Resolve Apply a function on every element of a stream.
/**/
// Typically is not usefully, because "Map" and "Filter" cover almost
// all conventional conditions.
// It is somehow UNSAFE, because the real rtype of Invoke().result have not been checked which considering
// be the same as what IResolver's OutType() pointer out.
func (s *ObjectStream) Resolve(resolver IResolver) *ObjectStream {
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
	for s.iter.HasNext() {
		next := s.iter.Next()
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

func (m *mapperFun) Invoke(v r.Value) IResolveResult {
	return &resolveResult{result: m.f.Call([]r.Value{v})[0], valid: true}
}

type filterFun struct {
	f           r.Value
	contentType r.Type
}

func (f *filterFun) Invoke(v r.Value) IResolveResult {
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
	iter              IMapIterator
	entryResolveChain []IEntryResolver
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
		entryResolveChain: make([]IEntryResolver, 0, 0),
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

func (ms *MapEntryStream) FilterValue(filter Filter) *MapEntryStream {
	if ms.err != nil {
		return ms
	}
	// TODO do type check
	ms.entryResolveChain = append(ms.entryResolveChain, &entryValueFilter{
		f:                r.ValueOf(filter),
		contentKeyType:   ms.outKeyType(),
		contentValueType: ms.outValueType(),
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
	for ms.iter.HasNext() {
		next := ms.iter.Next()
		valid := true
		var entryValue IEntry = &entry{k: r.ValueOf(next.Key()), v: r.ValueOf(next.Value())}
		for _, fun := range ms.entryResolveChain {
			result := fun.Invoke(entryValue)
			if result.Ok() {
				entryValue = result.Result()
				continue
			}
			valid = false
			break
		}
		if valid {
			target.SetMapIndex(entryValue.Key(), entryValue.Value())
		}
	}
	return &collectResult{origin: target, v: target, err: nil}
}

/********************************************* entry resolve result ***************************************************/

type entryResolveResult struct {
	result IEntry
	ok     bool
}

func (e *entryResolveResult) Result() IEntry {
	return e.result
}

func (e *entryResolveResult) Ok() bool {
	return e.ok
}

type entry struct {
	k r.Value
	v r.Value
}

func (e *entry) Key() r.Value {
	return e.k
}

func (e *entry) Value() r.Value {
	return e.v
}

/********************************************* entry stream resolver **************************************************/

type entryValueFilter struct {
	f                r.Value
	contentKeyType   r.Type
	contentValueType r.Type
}

func (ef *entryValueFilter) Invoke(e IEntry) IEntryResolveResult {
	ok := ef.f.Call([]r.Value{e.Value()})[0].Bool()
	if ok {
		return &entryResolveResult{
			ok:     true,
			result: e,
		}
	}
	return &entryResolveResult{ok: false}
}

func (ef *entryValueFilter) OutKeyType() r.Type {
	return ef.contentKeyType
}

func (ef *entryValueFilter) OutValueType() r.Type {
	return ef.contentValueType
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
