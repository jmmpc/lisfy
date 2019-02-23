package main

import (
	"context"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/jmmpc/progressreader"
)

func indexHandler(filename string) http.HandlerFunc {
	index := template.Must(template.ParseFiles(filename))

	return func(w http.ResponseWriter, r *http.Request) {
		if pusher, ok := w.(http.Pusher); ok {
			push(pusher,
				"static/css/styles.css",
				"static/js/app.js",
				"static/img/file_24px.svg",
				"static/img/folder_24px.svg",
				"static/img/refresh_white.svg",
				"static/img/arrow_back_white.svg",
			)
		}

		if err := index.Execute(w, r.URL.Path); err != nil {
			http.Error(w, "Something went wrong. Try again later.", http.StatusInternalServerError)
		}
	}
}

func dirHandler(dirname string) http.HandlerFunc {
	if !isExist(dirname) {
		log.Fatalf("dir %s is not exist", dirname)
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if containsDotDot(r.URL.Path) {
			http.Error(w, "invalid URL path", http.StatusBadRequest)
			return
		}

		filename := filepath.Join(dirname, filepath.FromSlash(r.URL.Path))

		stat, err := os.Stat(filename)
		switch {
		case os.IsNotExist(err):
			log.Printf("failed to read file stat: %v\n", err)
			http.Error(w, err.Error(), http.StatusNoContent)
		case os.IsPermission(err):
			log.Printf("failed to read file stat: %v\n", err)
			http.Error(w, err.Error(), http.StatusForbidden)
		case stat.IsDir():
			stats, _ := readdir(filename)
			if err := respondWithJSON(w, stats); err != nil {
				log.Printf("failed to marshal json: %v\n", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		case stat.Mode().IsRegular():
			if err := respondWithJSON(w, Marshaler(stat)); err != nil {
				log.Printf("failed to marshal json: %v\n", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		default:
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
	}
}

func uploadHandler(root string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if containsDotDot(r.URL.Path) {
			http.Error(w, "invalid URL path", http.StatusBadRequest)
			return
		}

		filename := makeUnique(filepath.Join(root, filepath.FromSlash(r.URL.Path)))

		file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if err != nil {
			log.Println("failed to create file: ", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		p := progressreader.WithContext(r.Context(), r.Body)

		_, err = io.Copy(file, p)

		if err := file.Close(); err != nil {
			log.Printf("failed to close file: %v\n", err)
		}

		switch err {
		case nil:
			dir, name := filepath.Split(filename)
			w.WriteHeader(http.StatusOK)
			log.Printf("file \"%s\" received from %s and saved to \"%s\"\n", name, r.RemoteAddr, dir)
		case context.Canceled, io.ErrUnexpectedEOF:
			os.Remove(filename)
			log.Printf("%s transfer is canceled: %v\n", filepath.Base(filename), err)
			http.Error(w, "file transfer is canceled", http.StatusInternalServerError)
		default:
			os.Remove(filename)
			log.Println("failed to save file: ", err)
			http.Error(w, "failed to save file", http.StatusInternalServerError)
		}
	}
}

func serveFile(root string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filename := filepath.Join(root, filepath.FromSlash(r.URL.Path))
		log.Printf("Client %s requested the file \"%s\"\n", r.RemoteAddr, filename)
		if dir, err := isdir(filename); dir || err != nil {
			http.Error(w, "no such file", http.StatusNotFound)
			return
		}
		http.ServeFile(w, r, filename)
	}
}
