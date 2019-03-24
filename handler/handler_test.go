package handler

import (
	"io/ioutil"
	"os"
	"testing"
)

const path = `C:\Users\Victor`

// func TestReadDir2(t *testing.T) {
// 	fis1, _ := readdir(path)
// 	fis2, _ := readdir2(path)
// 	if fis1 != fis2 {
// 		t.Errorf("wrong output")
// 	}
// }

var benchmarks = []struct {
	name string
	f    func(string) ([]os.FileInfo, error)
}{
	{"1", readdir},
	{"2", ioutil.ReadDir},
}

func BenchmarkReadDir(b *testing.B) {
	for _, benchmark := range benchmarks {
		b.Run(benchmark.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				benchmark.f(path)
			}
		})
	}
}
