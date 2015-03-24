package main

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/hashicorp/mdns"
)

func publish() {
	host, err := os.Hostname()
	if err != nil {
		fmt.Fprintf(os.Stderr, "hostname: %v\n", err)
		return
	}

	port, bs, err := startpublisher(host)
	if err != nil {
		fmt.Fprintf(os.Stderr, "startpublisher: %v\n", err)
		return
	}

	ip, err := getIP()
	if err != nil {
		fmt.Fprintf(os.Stderr, "getIP: %v\n", err)
		return
	}

	service, err := mdns.NewMDNSService(
		host,
		"_airlift._tcp",
		"",
		"",
		port,
		[]net.IP{ip},
		[]string{"airlift"},
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "mdns newservice: %v\n", err)
		return
	}

	server, _ := mdns.NewServer(&mdns.Config{Zone: service})
	defer server.Shutdown()

	if _, err := io.Copy(bs, os.Stdin); err != nil {
		fmt.Fprintf(os.Stderr, "io copy: %v\n", err)
		return
	}

	if err := bs.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "byteserver close: %v\n", err)
		return
	}
}

func validIFName(name string) bool {
	if name == "lo" {
		return false
	}
	if strings.HasPrefix(name, "docker") {
		return false
	}
	if strings.HasPrefix(name, "vbox") {
		return false
	}
	return true
}

func getIP() (net.IP, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	for _, iface := range ifaces {
		if validIFName(iface.Name) {
			addrs, err := iface.Addrs()
			if err != nil {
				return nil, err
			}

			for _, addr := range addrs {
				if strings.Contains(addr.String(), ".") {
					ip, _, err := net.ParseCIDR(addr.String())
					return ip, err
				}
			}
		}
	}

	return nil, nil
}

type byteserver struct {
	l  net.Listener
	c  net.Conn
	ch struct {
		pass chan struct{}
		fail chan struct{}
	}
}

func (bs *byteserver) wait() bool {
	select {
	case <-bs.ch.pass:
		return true
	case <-bs.ch.fail:
		return false
	}
}

func (bs *byteserver) Write(b []byte) (int, error) {
	if !bs.wait() {
		return 0, errors.New("accept error")
	}
	return bs.c.Write(b)
}

func (bs *byteserver) Close() error {
	if !bs.wait() {
		return errors.New("accept error")
	}
	return bs.c.Close()
}

func startpublisher(host string) (int, io.WriteCloser, error) {
	l, err := net.Listen("tcp", fmt.Sprintf("%s:0", host))
	if err != nil {
		return 0, nil, err
	}

	_, ps, err := net.SplitHostPort(l.Addr().String())
	if err != nil {
		return 0, nil, err
	}

	port, err := strconv.Atoi(ps)
	if err != nil {
		return 0, nil, err
	}

	bs := &byteserver{l: l}
	bs.ch.pass = make(chan struct{})
	bs.ch.fail = make(chan struct{})

	go func() {
		c, err := l.Accept()
		if err != nil {
			close(bs.ch.fail)
		} else {
			bs.c = c
			close(bs.ch.pass)
		}
	}()

	return port, bs, nil
}
