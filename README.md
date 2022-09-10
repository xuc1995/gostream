# gostream
A go module supplying Java-Like generic stream programming (while do type check at runtime)

## Using
### Get a Stream
To get a Stream, using gostream.SliceStream(*yourSlice*) or gostream.S(*yourSlice*) for short.  
To get a EntryStream, using gostream.EntrySlice or gostream.ES.  

### Using the Stream
At that time point, docs are rare, check *_test.go to figure out how to use it

#### simple examples
```go
package main

import (
	"github/xuc1995/gostream"
	"log"
)


type Person struct {
	name string
	age int
}

func main() {
	personList := []*Person {
		{name: "YuCong Xu", age: 24},
		{name: "Rui Li", age: 20},
		{name: "Sad Dog", age: 18},
		{name: "Lyric", age: 21},
	}
	var adultNames []string
	err := gostream.S(personList).
		Filter(func(p *Person) bool { return p.age > 18}).
		Map(func(p *Person) string {return p.name}).
		CollectAt(&adultNames)
	if err != nil {
		log.Panic(err.Error())
	}
}
```

## Warning
