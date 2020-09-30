package main

import (
	"flag"
	"fmt"
)

var (
	showVersion = flag.Bool("version", false, "show build version.")
)

func main() {
	fmt.Printf("hello")
}
