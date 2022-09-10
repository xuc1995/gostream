package main

import (
	"github.com/xuc1995/gostream"
	"log"
)

type Person struct {
	name string
	age  int
}

func main() {
	personList := []*Person{
		{name: "YuCong Xu", age: 24},
		{name: "Rui Li", age: 20},
		{name: "Sad Dog", age: 18},
		{name: "Lyric", age: 21},
	}
	var adultNames []string
	err := gostream.S(personList).
		Filter(func(p *Person) bool { return p.age > 18 }).
		Map(func(p *Person) string { return p.name }).
		CollectAt(&adultNames)
	if err != nil {
		log.Panic(err.Error())
	}
}
