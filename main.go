package main

import (
	"flag"
	"os"

	"github.com/andrew-d/go-termutil"
)

var (
	pubOnly  = flag.Bool("p", false, "only publish, regardless of TTY stdin")
	recvOnly = flag.Bool("r", false, "only receive, regardless of TTY stdin")
)

func main() {
	flag.Parse()

	switch {
	case *recvOnly:
		receive()
	case *pubOnly:
		publish()
	case termutil.Isatty(os.Stdin.Fd()):
		receive()
	default:
		publish()
	}
}
