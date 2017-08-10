package nexus

import (
	"archive/zip"
	"bytes"
	"fmt"
	"github.com/docker/docker/pkg/homedir"
	"github.com/skatteetaten/architect/pkg/config"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDownloadFromNexusServer(t *testing.T) {
	b, err := createZipFile()
	if err != nil {
		t.Error(err.Error())
	}
	zipFileName := "my-test-package-1.0.0-Leveransepakke.zip"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(b.Bytes())))
		w.Header().Set("Content-Disposition", "attachment; filename=\""+zipFileName+"\"")
		w.Header().Set("Content-Type", "application/zip")
		w.Write(b.Bytes())
	}))
	defer ts.Close()

	n := NewNexusDownloader(ts.URL)
	m := config.JavaApplication{
		ArtifactId: "openshift-resource-monitor",
		GroupId:    "ske.fellesplattform.monitor",
		Version:    "1.1.4",
	}

	r, err := n.DownloadArtifact(&m)
	if err != nil {
		t.Error(err.Error())
	}

	if strings.Contains(r.Path, zipFileName) == false {
		t.Error(
			"excpected", zipFileName,
			"got", r)
	}
}

func TestNewLocalDownloader(t *testing.T) {
	d := NewLocalDownloader()
	m := config.JavaApplication{
		ArtifactId: "dontexist",
		GroupId:    "ske",
		Version:    "develop-SNAPSHOT",
	}
	l, err := d.DownloadArtifact(&m)
	homedir := homedir.Get()
	expected := homedir + "/.m2/repository/ske/dontexist/develop-SNAPSHOT/" +
		"dontexist-develop-SNAPSHOT-Leveransepakke.zip"
	if l.Path != expected {
		t.Errorf("Expexted %s, was %s", expected, l)
	}
	if err == nil {
		t.Error("Error expected on non existing file")
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
