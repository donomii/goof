// demo.go
package main

import (
	"fmt"

	".."
)

func main() {
	goof.QC([]string{"ulimit", "-n", "99999"})
	res := goof.WrappedTraceroute("8.8.8.8")
	fmt.Println(res)
}
