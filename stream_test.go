package gostream

import (
	"fmt"
	r "reflect"
	"strconv"
	"testing"
)

func Test_Typical_SliceStream(t *testing.T) {
	target := []int{1, 4, 9, 16, 36}
	slice, err := SliceStream([]string{"1", "2", "3", "4", "55", "6"}).
		Filter(func(it string) bool { return len(it) < 2 }).
		Map(func(it string) int {
			i, err := strconv.Atoi(it)
			if err != nil {
				i = 0
			}
			return i * i
		}).
		Collect()
	if err != nil {
		panic(err)
	}
	if !r.DeepEqual(target, slice) {
		panic(fmt.Sprintf("target slice is: %v, while get: %v\n", target, slice))
	}
}

type doubleIntResult struct {
	result r.Value
}

func (dr *doubleIntResult) Result() r.Value {
	return dr.result
}

func (dr *doubleIntResult) Ok() bool {
	return true
}

// multiple an int (to int64 because of reflecting)
type multiInt struct {
	fac int
}

func (dr *multiInt) Invoke(v r.Value) IResolveResult {
	return &doubleIntResult{result: r.ValueOf(v.Int() * int64(dr.fac))}
}

// Representing int time during reflecting
var zero int64

func (dr *multiInt) OutType() r.Type {
	return r.TypeOf(zero)
}

func Test_Custom_Resolver(t *testing.T) {
	s := []int{-1, 0, 1, 2, 3, 4, 5}
	target := []int64{3, 6, 9, 12, 15, 18}

	var slice []int64
	err := SliceStream(s).
		Map(func(it int) int { return it + 1 }).
		Filter(func(it int) bool { return it > 0 }).
		Resolve(&multiInt{3}). // object becomes int64 after that
		CollectAt(&slice)
	if err != nil {
		panic(err)
	}
	if !r.DeepEqual(slice, target) {
		panic(fmt.Sprintf("target slice is :%v, while get: %v\n", target, slice))
	}
	// TODO Add case
}

func Test_Typical_EntryStream(t *testing.T) {
	m := map[int]string{0: "0", 1: "1", 2: "2", 3: "3", 4: "4", 5: "5", 99: "99"}
	fvTarget := map[int]string{99: "99"}
	fvRes := make(map[int]string)

	err := EntryStream(m).
		FilterValue(func(it string) bool { return len(it) > 1 }).
		CollectAt(&fvRes)
	if err != nil {
		panic(err)
	}
	if !r.DeepEqual(fvRes, fvTarget) {
		panic(fmt.Sprintf("target result after filter-value is: %v, while get: %v\n", fvTarget, fvRes))
	}
	// TODO Add case
}

func Test_StreamToMap(t *testing.T) {
	s := []int{0, 1, 2, 3, 4, 5, 6}
	targetMap := map[int]string{0: "0", 1: "1", 2: "2", 3: "3", 4: "4", 5: "5"}
	res, err := SliceStream(s).
		AsMapKey(func(it int) string { return strconv.Itoa(it) }).
		FilterValue(func(it string) bool { return it != "6" }).
		Collect()
	if err != nil {
		panic(err)
	}
	if !r.DeepEqual(targetMap, res) {
		panic(fmt.Sprintf("target map is: %v while get: %v\n", targetMap, res))
	}
}

// TODO Add failed condition test
