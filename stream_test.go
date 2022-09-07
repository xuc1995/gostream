package gostream

import (
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
		t.Error(err)
	}
	if !r.DeepEqual(target, slice) {
		t.Errorf("target sequence is: %v, while get: %v\n", target, slice)
	}
}

func Test_Typical_EntryStream(t *testing.T) {
	m := map[int]string{0: "0", 1: "1", 2: "2", 3: "3", 4: "4", 5: "5", 99: "99"}
	fvTarget := map[int]string{99: "99"}
	fvRes := make(map[int]string)

	err := EntryStream(m).
		Filter(func(k int, v string) bool { return len(v) > 1 }).
		CollectAt(&fvRes)
	if err != nil {
		t.Error(err)
	}
	if !r.DeepEqual(fvRes, fvTarget) {
		t.Errorf("target result after filter-value is: %v, while get: %v\n", fvTarget, fvRes)
	}
	// TODO Add case
}

func Test_StreamToMap(t *testing.T) {
	s := []int{0, 1, 2, 3, 4, 5, 6}
	targetMap := map[int]string{0: "0", 1: "1", 2: "2", 3: "3", 4: "4", 5: "5"}
	res, err := SliceStream(s).
		ToEntryStream(func(it int) (int, string) { return it, strconv.Itoa(it) }).
		Filter(func(k int, v string) bool { return v != "6" }).
		Collect()
	if err != nil {
		t.Error(err)
	}
	if !r.DeepEqual(targetMap, res) {
		t.Errorf("target map is: %v while get: %v\n", targetMap, res)
	}
}

type intGenerator0to10 int

func (t *intGenerator0to10) Next() bool {
	hasNext := *t < 10
	*t++
	return hasNext
}

func (t *intGenerator0to10) Value() interface{} {
	return int(*t)
}

func Test_CustomIterator(t *testing.T) {
	var collectAt []int
	generator0to10 := intGenerator0to10(-1)
	err := Stream(&generator0to10, 0).Filter(func(it int) bool {
		return it&1 == 0
	}).CollectAt(&collectAt)
	if err != nil {
		t.Error(err)
	}
	target := []int{0, 2, 4, 6, 8, 10}
	if !r.DeepEqual(target, collectAt) {
		t.Errorf("filter result should be: %v while is: %v", target, collectAt)
	}
}

// TODO Add failed condition test
