package main

import (
	"encoding/json"
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

func (f FileInfo) MarshalJSON() ([]byte, error) {
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

func Marshaler(info os.FileInfo) json.Marshaler {
	return FileInfo{info}
}

func readDirMap(dirname string, mapping func(info os.FileInfo) bool) ([]*FileInfo, error) {
	f, err := os.Open(dirname)
	if err != nil {
		return nil, err
	}
	list, _ := f.Readdir(-1)
	if err := f.Close(); err != nil {
		return nil, err
	}

	var fileInfoList []*FileInfo
	for _, info := range list {
		if mapping(info) {
			fileInfoList = append(fileInfoList, &FileInfo{info})
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
	return readDirMap(dirname, func(info os.FileInfo) bool {
		if len(info.Name()) != 0 && info.Name()[0] == '.' {
			return false
		}
		if !info.Mode().IsRegular() && !info.IsDir() {
			return false
		}
		return true
	})
}

// func ReadDir(dirname string) ([]*FileInfo, error) {
// 	f, err := os.Open(dirname)
// 	if err != nil {
// 		return nil, err
// 	}
// 	list, err := f.Readdir(-1)
// 	f.Close()
// 	if err != nil {
// 		return nil, err
// 	}

// 	fileInfoList := make([]*FileInfo, len(list))
// 	for i, info := range list {
// 		fileInfoList[i] = &FileInfo{info}
// 	}

// 	return fileInfoList, err
// }

// func ReadDirFunc(dirname string, f func(info os.FileInfo) bool) ([]*FileInfo, error) {
// 	fis, err := ReadDir(dirname)
// 	if err != nil {
// 		return nil, err
// 	}

// 	var l []*FileInfo
// 	for _, fi := range fis {
// 		if f(fi) {
// 			l = append(l, fi)
// 		}
// 	}
// 	return l, nil
// }

// push used for http/2 responses.
func push(pusher http.Pusher, resources ...string) {
	for _, res := range resources {
		pusher.Push(res, nil)
	}
}

// func readdir(dirname string) ([]*FileInfo, error) {
// 	fis, err := ReadDirFunc(dirname, func(info os.FileInfo) bool {
// 		if len(info.Name()) != 0 && info.Name()[0] == '.' {
// 			return false
// 		}
// 		if !info.Mode().IsRegular() && !info.IsDir() {
// 			return false
// 		}
// 		return true
// 	})
// 	if err != nil {
// 		return nil, err
// 	}

// 	sort.SliceStable(fis, func(i, j int) bool {
// 		if fis[i].IsDir() && !fis[j].IsDir() {
// 			return true
// 		} else if !fis[i].IsDir() && fis[j].IsDir() {
// 			return false
// 		}
// 		// return strings.ToLower(fis[i].Name()) < strings.ToLower(fis[j].Name())
// 		return less(fis[i].Name(), fis[j].Name())
// 	})

// 	return fis, nil
// }

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

func isExist(filename string) bool {
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

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP, nil
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
	if home, ok := os.LookupEnv("HOME"); ok {
		return home
	}
	return "."
}

func isdir(name string) (bool, error) {
	info, err := os.Stat(name)
	if err != nil {
		return false, err
	}

	if info.IsDir() {
		return true, nil
	}

	return false, nil
}
