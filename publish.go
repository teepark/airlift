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

	ips, err := getIPs()
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
		ips,
		nil,
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

func getIPs() ([]net.IP, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	ip4s := make([]net.IP, 0)
	ips := make([]net.IP, 0)

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			// interface down
			continue
		}
		if iface.Flags&net.FlagLoopback != 0 {
			// loopback interface
			continue
		}
		if strings.HasPrefix(iface.Name, "docker") {
			// docker0 -- needs special casing, can't figure out
			// what's so special about it
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			return nil, err
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if ip == nil || ip.IsLoopback() {
				continue
			}

			ips = append(ips, ip)
			if ip.To4() != nil {
				ip4s = append(ip4s, ip)
			}
		}
	}

	if len(ip4s) != 0 {
		ips = ip4s
	}
	return ips, nil
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
