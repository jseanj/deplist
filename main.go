package main

import (
	"log"
	"os"
	"sort"
	//"strings"
	"flag"
)

func must(value interface{}, err error) interface{} {
	return value
}

var path = flag.String("path", must(os.Getwd()).(string), "project path")
var verbose = flag.Bool("v", false, "verbose")

func main() {
	flag.Parse()
	deps, err := List(*path, os.Args[1:]...)
	if err != nil {
		log.Fatal(err)
	}
	list := make([]string, 0, len(deps))
	for k := range deps {
		list = append(list, k)
	}
	sort.Strings(list)
	for _, url := range list {
		GetDetail(url)
	}
	//fmt.Println(strings.Join(list, "\n"))
}
