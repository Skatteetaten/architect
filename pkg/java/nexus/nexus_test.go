package nexus

import (
	"testing"
	"fmt"
	"net/http/httptest"
	"net/http"
	"bytes"
	"archive/zip"
	"log"
	"strings"
)

func getArtifact(d Downloader) (string, error) {
	return d.DownloadArtifact()
}

func TestNexus(t *testing.T) {
	b, err := createZipFile()
	if err != nil {
		t.Error(err.Error())
	}
	zipFileName := "my-test-package-1.0.0-Leveransepakke.zip"

	ts := httptest.NewServer(http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d",len(b.Bytes())))
		w.Header().Set("Content-Disposition", "attachment; filename=\"" + zipFileName + "\"")
		w.Header().Set("Content-Type", "application/zip")
		w.Write(b.Bytes())
	}))
	defer ts.Close()

	n := &Nexus{BaseUrl:ts.URL, ArtifactId:"openshift-resource-monitor", GroupId:"ske.fellesplattform.monitor", Version:"1.1.4", Type:"zip"}
	r, err := getArtifact(n);
	if err != nil {
		t.Error(err.Error())
	}

	if strings.Contains(r, zipFileName) == false {
		t.Error(
			"excpected", zipFileName,
			"got", r)
	}
}

func createZipFile() (bytes.Buffer, error) {
	buf := new(bytes.Buffer)

	z := zip.NewWriter(buf)
	var files = []struct {
		Name, Body string
	}{
		{"somefile", "this file is for testing"},
	}
	for _, file := range files {
		f, err := z.Create(file.Name)
		if err != nil {
			return *buf, err
		}
		_, err = f.Write([]byte(file.Body))
		if err != nil {
			return *buf, err
		}
	}
	err := z.Close()
	if err != nil {
		log.Fatal(err)
	}
	return *buf, nil
}