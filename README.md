# gostream
A go module supplying Java-Like generic stream programming (while do type check at runtime)

## Using
### Get a Stream
To get a Stream, using SliceStream(yourSlice).
To get a EntryStream in which map key-value entry, useing EntryStream(youMap)

### Using the Stream
Because at that time point, the doc is very limited, check *_test.go to figure out how to use it

#### simple examples
```go
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
```

## Warning
At that point, runtime type check haven't been implemented, as to say, it will PANIC if you pass a parameter with a wrong type (as it is held as interface{}, it will not be checked and compile time).

I am going to add runtime type check next step, so it may return error instead of just PANIC in the nearly future.
