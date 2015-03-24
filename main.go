package main

import (
	"os"

	"github.com/andrew-d/go-termutil"
)

func main() {
	if termutil.Isatty(os.Stdin.Fd()) {
		receive()
	} else {
		publish()
	}
}
