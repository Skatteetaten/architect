package docker

import (
	"net/http/httptest"
	"io/ioutil"
	"net/http"
	"fmt"
)

func StartMockRegistry() (*httptest.Server, error) {
	buf, err := ioutil.ReadFile("testdata/manifest.json")

	if err != nil {
		return nil, err
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(buf)))
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Write(buf)
	}))

	return ts, nil
}
