package main

import (
	"architect/pkg/java/nexus"
	"architect/pkg/java/prepare"
	"fmt"
)

func main() {
	n := &nexus.Nexus{}
	fmt.Printf("%+v", n)
	prepare.Prepare()
}