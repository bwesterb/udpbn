package main

import (
	"flag"
	"log"
	"net"
	"time"
)

var (
	listenAddr   = flag.String("listen", ":7878", "listen op given port")
	upstreamAddr = flag.String("upstream", "127.0.0.1:7879",
		"proxy to given upstream address")
	rate       = flag.Float64("rate", 100000, "limit to given bytes/sec")
	lastSecond int64
	dataUp     uint64
	dataDown   uint64
	addr       net.Addr
)

func allow(amount uint64, up bool) bool {
	now := time.Now().UnixNano()
	secondNow := now / 1000000000
	if secondNow != lastSecond {
		dataUp = 0
		dataDown = 0
		lastSecond = secondNow
	} else {
		var data uint64
		if up {
			data = dataUp
		} else {
			data = dataDown
		}
		if *rate < float64(data)/(float64(now%1000000000)/1000000000) {
			return false
		}
	}
	if up {
		dataUp += amount
	} else {
		dataDown += amount
	}
	return true
}

func main() {
	flag.Parse()
	ls, err := net.ListenPacket("udp", *listenAddr)
	if err != nil {
		log.Fatalf("ListenPacket: %v", err)
	}
	us, err := net.Dial("udp", *upstreamAddr)
	if err != nil {
		log.Fatalf("Dial: %v", err)
	}

	go func() {
		buf := make([]byte, 1500)
		for {
			n, addr2, err := ls.ReadFrom(buf[:])
			if !allow(uint64(n), true) {
				continue
			}
			addr = addr2
			_, err2 := us.Write(buf[:n])
			if err != nil {
				log.Printf("ReadFrom: %v", err)
			}
			if err2 != nil {
				log.Printf("Write: %v", err2)
			}
		}
	}()
	buf := make([]byte, 1500)
	for {
		n, err := us.Read(buf[:])
		if addr == nil {
			log.Printf("Don't know where to send this packet to ...")
			continue
		}
		if !allow(uint64(n), false) {
			continue
		}
		_, err2 := ls.WriteTo(buf[:n], addr)
		if err != nil {
			log.Printf("Read: %v", err)
		}
		if err2 != nil {
			log.Printf("WriteTo: %v", err2)
		}
	}
}
