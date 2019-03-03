package main

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

type FileInfo struct {
	os.FileInfo
}

func (f *FileInfo) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Name    string      `json:"name"`
		Size    int64       `json:"size"`
		IsDir   bool        `json:"isdir"`
		ModTime int64       `json:"modtime"`
		Mode    os.FileMode `json:"mode"`
	}{
		Name:    f.Name(),
		Size:    f.Size(),
		IsDir:   f.IsDir(),
		ModTime: f.ModTime().UnixNano(),
		Mode:    f.Mode(),
	})
}

func readDirMap(dirname string, mapping func(fi os.FileInfo) bool) ([]*FileInfo, error) {
	f, err := os.Open(dirname)
	if err != nil {
		return nil, err
	}
	fis, err := f.Readdir(-1)
	f.Close()

	if err != nil && len(fis) == 0 {
		return nil, err
	}

	var fileInfoList []*FileInfo
	for _, fi := range fis {
		if mapping(fi) {
			fileInfoList = append(fileInfoList, &FileInfo{fi})
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

func readdir(dirname string) ([]*FileInfo, error) {
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

func exist(filename string) bool {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return false
	}
	return true
}

// makeUnique adds current time to file name.
func makeUnique(filename string) string {
	ext := filepath.Ext(filename)
	name := strings.TrimSuffix(filename, ext)
	return name + "_" + time.Now().Format("2006-01-02_150405") + ext
}

// localIP returns the string form of the current device local ip address and error if any.
func localIP() (net.IP, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	addr := conn.LocalAddr().(*net.UDPAddr)

	return addr.IP, nil
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

func homedir() string {
	if home, err := os.UserHomeDir(); err == nil {
		return home
	}
	return "."
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
