package main

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

const (
	jsonContentType = "application/json; charset=utf-8"
	htmlContentType = "text/html; charset=utf-8"
)

var methods = []string{http.MethodConnect, http.MethodDelete, http.MethodGet,
	http.MethodHead, http.MethodOptions, http.MethodPatch,
	http.MethodPost, http.MethodPut, http.MethodTrace}

func TestIndexPage(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	rr := httptest.NewRecorder()

	indexHandler("index.html").ServeHTTP(rr, req)

	resp := rr.Result()

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("status code differs. Expected %d. Got %d", http.StatusOK, status)
	}

	if contentType := resp.Header.Get("Content-Type"); contentType != htmlContentType {
		t.Errorf("content type differs. Expected %s. Got %s", htmlContentType, contentType)
	}

	body, _ := ioutil.ReadAll(resp.Body)
	if len(body) == 0 {
		t.Errorf("response body must not be empty")
	}
	resp.Body.Close()
}

func TestDirHandler(t *testing.T) {
	tt := []struct {
		name        string
		path        string
		err         string
		status      int
		contentType string
	}{
		{name: "no such dir", path: "/djfeuhws", err: "internal server error", status: http.StatusInternalServerError, contentType: "text/plain"},
		{name: "success", path: "/Downloads", err: "", status: http.StatusOK, contentType: jsonContentType},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", tc.path, nil)
			if err != nil {
				t.Fatalf("could not create request: %v", err)
			}

			rr := httptest.NewRecorder()
			dirHandler("/home/victor/").ServeHTTP(rr, req)

			res := rr.Result()
			defer res.Body.Close()

			if rr.Code != tc.status {
				t.Errorf("expected status %d; got %d", tc.status, rr.Code)
			}

			body, err := ioutil.ReadAll(res.Body)
			if err != nil {
				t.Fatalf("could not read response: %v", err)
			}

			if tc.err != "" {
				if msg := string(bytes.TrimSpace(body)); msg != tc.err {
					t.Errorf("expected message %q; got %q", tc.err, msg)
				}
				return
			}

			if contentType := res.Header.Get("Content-Type"); contentType != tc.contentType {
				t.Errorf("expected Content-Type %s; got %s", tc.contentType, contentType)
			}
		})
	}
}

func TestMethods(t *testing.T) {
	ts := httptest.NewServer(newServer(":8080", "/home/victor/").router)
	defer ts.Close()

	testMethods(t, ts.URL+"/", http.MethodGet)
	testMethods(t, ts.URL+"/files/", http.MethodGet)
	testMethods(t, ts.URL+"/upload/", http.MethodPost)
	testMethods(t, ts.URL+"/static/", http.MethodGet)
	testMethods(t, ts.URL+"/download/", http.MethodGet)
}

func testMethods(t *testing.T, url string, allowedMethods ...string) {
	for _, allowedMethod := range allowedMethods {
		for _, method := range methods {
			req, err := http.NewRequest(method, url, nil)
			if err != nil {
				t.Fatalf("could not create request: %v", err)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}

			if method != allowedMethod && resp.StatusCode != http.StatusMethodNotAllowed {
				t.Errorf("rout %s accepts wrong method: allowed methods are %v; accepted method is %s", url, allowedMethods, method)
			}

			if method == allowedMethod && resp.StatusCode == http.StatusMethodNotAllowed {
				t.Errorf("rout %s do not accepts allowed method %s", url, method)
			}
		}
	}
}
