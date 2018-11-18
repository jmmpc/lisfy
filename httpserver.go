package main

import (
	"log"
	"net/http"
	"path/filepath"

	"github.com/NYTimes/gziphandler"
	"github.com/gorilla/mux"
)

type server struct {
	router *mux.Router
	root   string
	addr   string
}

func newServer(addr, root string) *server {
	s := &server{
		router: mux.NewRouter(),
		root:   root,
		addr:   addr,
	}
	s.routes()
	return s
}

func (s *server) routes() {
	s.router.Handle("/", gziphandler.GzipHandler(indexHandler("index.html"))).Methods("GET")
	s.router.PathPrefix("/files/").Handler(http.StripPrefix("/files/", dirHandler(s.root))).Methods("GET")
	s.router.PathPrefix("/download/").Handler(http.StripPrefix("/download/", serveFile(s.root))).Methods("GET")
	s.router.PathPrefix("/static/").Handler(http.FileServer(http.Dir("."))).Methods("GET")
	s.router.PathPrefix("/upload/").Handler(http.StripPrefix("/upload/", uploadHandler(s.root))).Methods("POST")
}

func (s *server) run() error {
	root, err := filepath.Abs(s.root)
	if err != nil {
		return err
	}

	log.Printf("Serving %s\n", root)
	return http.ListenAndServe(s.addr, s.router)
}
