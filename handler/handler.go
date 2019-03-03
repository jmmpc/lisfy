package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	gzp "github.com/NYTimes/gziphandler"
	"github.com/gorilla/mux"
	"github.com/jmmpc/progressreader"
)

type handlerFunc func(http.ResponseWriter, *http.Request) (status int, err error)

func (h handlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	status, err := h(w, r)
	if err != nil {
		http.Error(w, err.Error(), status)
	}
}

type handler struct {
	root      string
	indexHTML string
}

// New returns an http handler.
func New(root string, indexHTML string) http.Handler {
	h := handler{root, indexHTML}
	router := mux.NewRouter()
	router.Handle("/", gzp.GzipHandler(handlerFunc(h.indexHandler))).Methods("GET")
	router.PathPrefix("/files/").Handler(http.StripPrefix("/files/", handlerFunc(h.dirHandler))).Methods("GET")
	router.PathPrefix("/download/").Handler(http.StripPrefix("/download/", http.HandlerFunc(h.serveFile))).Methods("GET")
	router.PathPrefix("/static/").Handler(http.FileServer(http.Dir("."))).Methods("GET")
	router.PathPrefix("/upload/").Handler(http.StripPrefix("/upload/", handlerFunc(h.uploadHandler))).Methods("POST")
	return router
}

func (h handler) indexHandler(w http.ResponseWriter, r *http.Request) (int, error) {
	index := template.Must(template.ParseFiles(h.indexHTML))

	if pusher, ok := w.(http.Pusher); ok {
		options := &http.PushOptions{
			Header: http.Header{
				"Accept-Encoding": r.Header["Accept-Encoding"],
			},
		}
		push(pusher, options,
			"static/css/styles.css",
			"static/js/app.js",
			"static/img/file_24px.svg",
			"static/img/folder_24px.svg",
			"static/img/refresh_white.svg",
			"static/img/arrow_back_white.svg",
			"static/img/add_a_photo-24px.svg",
		)
	}

	if err := index.Execute(w, r.URL.Path); err != nil {
		log.Printf("Could not execute template: %v\n", err)
		return http.StatusInternalServerError, fmt.Errorf("something went wrong, try again later")
	}
	return http.StatusOK, nil
}

func (h handler) dirHandler(w http.ResponseWriter, r *http.Request) (int, error) {
	if containsDotDot(r.URL.Path) {
		return http.StatusBadRequest, fmt.Errorf("invalid URL path")
	}

	filename := filepath.Join(h.root, filepath.FromSlash(r.URL.Path))

	stat, err := os.Stat(filename)
	switch {
	case os.IsNotExist(err):
		log.Printf("failed to read file stat: %v\n", err)
		return http.StatusNoContent, fmt.Errorf("no such file or directory")
	case os.IsPermission(err):
		log.Printf("failed to read file stat: %v\n", err)
		return http.StatusForbidden, fmt.Errorf("permission denied")
	case stat.IsDir():
		stats, _ := readdir(filename)
		if err := respondWithJSON(w, stats); err != nil {
			log.Printf("failed to marshal json: %v\n", err)
			return http.StatusInternalServerError, fmt.Errorf("internal server error")
		}
	case stat.Mode().IsRegular():
		if err := respondWithJSON(w, &fileinfo{stat}); err != nil {
			log.Printf("failed to marshal json: %v\n", err)
			return http.StatusInternalServerError, fmt.Errorf("internal server error")
		}
	default:
		return http.StatusInternalServerError, fmt.Errorf("internal server error")
	}

	return http.StatusOK, nil
}

func (h handler) uploadHandler(w http.ResponseWriter, r *http.Request) (int, error) {
	if containsDotDot(r.URL.Path) {
		return http.StatusBadRequest, fmt.Errorf("invalid URL path")
	}

	filename := makeUnique(filepath.Join(h.root, filepath.FromSlash(r.URL.Path)))

	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		log.Println("failed to create file: ", err)
		return http.StatusInternalServerError, fmt.Errorf("internal server error")
	}

	pr := progressreader.WithContext(r.Context(), r.Body)

	_, err = io.Copy(file, pr)

	if err := file.Close(); err != nil {
		log.Printf("failed to close file: %v\n", err)
	}

	switch err {
	case nil:
		dir, name := filepath.Split(filename)
		log.Printf("file \"%s\" received from %s and saved to \"%s\"\n", name, r.RemoteAddr, dir)
		return http.StatusOK, nil
	case context.Canceled, io.ErrUnexpectedEOF:
		os.Remove(filename)
		log.Printf("%s transfer is canceled: %v\n", filepath.Base(filename), err)
		return http.StatusInternalServerError, fmt.Errorf("file transfer is canceled")
	default:
		os.Remove(filename)
		log.Println("failed to save file: ", err)
		return http.StatusInternalServerError, fmt.Errorf("internal server error")
	}
}

func (h handler) serveFile(w http.ResponseWriter, r *http.Request) {
	filename := filepath.Join(h.root, filepath.FromSlash(r.URL.Path))
	log.Printf("Client %s requested the file \"%s\"\n", r.RemoteAddr, filename)
	if dir, err := isdir(filename); dir || err != nil {
		http.Error(w, "no such file", http.StatusNoContent)
		return
	}
	http.ServeFile(w, r, filename)
}

// push used for http/2 responses.
func push(pusher http.Pusher, opts *http.PushOptions, resources ...string) {
	for _, res := range resources {
		err := pusher.Push(res, opts)
		if err != nil {
			log.Printf("Failed to push: %v", err)
		}
		if err == http.ErrNotSupported {
			return
		}
	}
}

func containsDotDot(v string) bool {
	if !strings.Contains(v, "..") {
		return false
	}
	for _, ent := range strings.FieldsFunc(v, isSlashRune) {
		if ent == ".." {
			return true
		}
	}
	return false
}

func isSlashRune(r rune) bool { return r == '/' || r == '\\' }

func respondWithJSON(w http.ResponseWriter, data interface{}) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	return json.NewEncoder(w).Encode(data)
}

type fileinfo struct {
	os.FileInfo
}

func (fi *fileinfo) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Name    string      `json:"name"`
		Size    int64       `json:"size"`
		IsDir   bool        `json:"isdir"`
		ModTime int64       `json:"modtime"`
		Mode    os.FileMode `json:"mode"`
	}{
		Name:    fi.Name(),
		Size:    fi.Size(),
		IsDir:   fi.IsDir(),
		ModTime: fi.ModTime().UnixNano(),
		Mode:    fi.Mode(),
	})
}

func readDirMap(dirname string, mapping func(fi os.FileInfo) bool) ([]*fileinfo, error) {
	f, err := os.Open(dirname)
	if err != nil {
		return nil, err
	}
	fis, err := f.Readdir(-1)
	f.Close()

	if err != nil && len(fis) == 0 {
		return nil, err
	}

	var fileInfoList []*fileinfo
	for _, fi := range fis {
		if mapping(fi) {
			fileInfoList = append(fileInfoList, &fileinfo{fi})
		}
	}

	sort.SliceStable(fileInfoList, func(i, j int) bool {
		if fileInfoList[i].IsDir() && !fileInfoList[j].IsDir() {
			return true
		} else if !fileInfoList[i].IsDir() && fileInfoList[j].IsDir() {
			return false
		}
		return less(fileInfoList[i].Name(), fileInfoList[j].Name())
	})

	return fileInfoList, nil
}

func readdir(dirname string) ([]*fileinfo, error) {
	return readDirMap(dirname, func(fi os.FileInfo) bool {
		if len(fi.Name()) != 0 && fi.Name()[0] == '.' {
			return false
		}
		if !fi.Mode().IsRegular() && !fi.IsDir() {
			return false
		}
		return true
	})
}

// less reports whether s1 should sort before s2.
func less(s1, s2 string) bool {
	for s1 != "" && s2 != "" {
		r1, size1 := utf8.DecodeRuneInString(s1)
		r2, size2 := utf8.DecodeRuneInString(s2)
		s1 = s1[size1:]
		s2 = s2[size2:]

		if r1 == r2 {
			continue
		}

		r1 = unicode.ToLower(r1)
		r2 = unicode.ToLower(r2)

		if r1 != r2 {
			return r1 < r2
		}
	}

	if s1 == "" && s2 != "" {
		return true
	}

	return false
}

// makeUnique adds current time to file name.
func makeUnique(filename string) string {
	ext := filepath.Ext(filename)
	name := strings.TrimSuffix(filename, ext)
	return name + "_" + time.Now().Format("2006-01-02_150405") + ext
}

func isdir(name string) (bool, error) {
	fi, err := os.Stat(name)
	if err != nil {
		return false, err
	}

	if fi.IsDir() {
		return true, nil
	}

	return false, nil
}
