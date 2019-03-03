package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"github.com/jmmpc/lisfy/handler"
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

	if !exist(*root) {
		log.Fatalf("%s is not exist\n", *root)
	}

	log.SetPrefix(">----------------------->\n")

	*root, err = filepath.Abs(*root)
	if err != nil {
		log.Fatalf("could not get absolute root path: %v\n", err)
	}

	log.Printf("Start serving files in %s\n", *root)

	server := &http.Server{
		Addr:    *addr,
		Handler: handler.New(*root, "index.html"),
	}

	if err := server.ListenAndServe(); err != nil {
		log.Printf("could not start server: %v\n", err)
	}
}

func localIP() (net.IP, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	addr := conn.LocalAddr().(*net.UDPAddr)

	return addr.IP, nil
}

func homedir() string {
	if home, err := os.UserHomeDir(); err == nil {
		return home
	}
	return "."
}

// exist reports whether file with provided name is exist
func exist(name string) bool {
	if _, err := os.Stat(name); os.IsNotExist(err) {
		return false
	}
	return true
}
