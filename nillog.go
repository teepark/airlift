package main

// github/hashicorp/mdns has log calls strewn all throughout,
// but we need our own use of stdout

import "log"

type nilout struct{}

func (n nilout) Write(b []byte) (int, error) {
	return len(b), nil
}

func init() {
	log.SetOutput(nilout{})
}
