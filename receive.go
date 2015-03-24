package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"strings"

	"github.com/hashicorp/mdns"
)

func receive() {
	entries := make(chan *mdns.ServiceEntry, 4)
	go func() {
		for entry := range entries {
			if strings.Contains(entry.Name, "_airlift._tcp") {
				if printFrom(entry) {
					os.Exit(0)
				}
			}
		}
	}()

	if err := mdns.Lookup("_airlift._tcp", entries); err != nil {
		fmt.Fprintf(os.Stderr, "mdns lookup: %v\n", err)
		return
	}
	close(entries)
}

func printFrom(entry *mdns.ServiceEntry) bool {
	ip := entry.AddrV4
	if ip == nil {
		ip = entry.AddrV6
	}
	if ip == nil {
		return false
	}

	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", ip.String(), entry.Port))
	if err != nil {
		return false
	}
	defer conn.Close()

	io.Copy(os.Stdout, conn)

	return true
}
