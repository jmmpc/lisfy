package main

import (
	"flag"
	"log"
)

func main() {
	addr := flag.String("http", ":7777", "listen address")
	root := flag.String("root", homedir(), "set file server root directory")

	flag.Parse()

	ip, err := localIP()
	if err != nil || len(ip) == 0 {
		log.Printf("Can not identify this device ip address: %v\n", err)
	} else {
		log.Printf("This device ip address: %s%s\n", ip, *addr)
	}

	if err := run(*addr, *root); err != nil {
		log.Printf("could not start server: %v\n", err)
	}
}
