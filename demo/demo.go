// demo.go
package main

import (
	"fmt"

	".."
)

func main() {
	res := goof.WrappedTraceroute("8.8.8.8")
	fmt.Println(res)
}
