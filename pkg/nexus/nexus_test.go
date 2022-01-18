package nexus

import (
	"archive/zip"
	"bytes"
	"fmt"
	"github.com/skatteetaten/architect/pkg/config"
	"github.com/stretchr/testify/assert"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestDownloadFromNexus2Server(t *testing.T) {
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

	n := NewNexusDownloader(ts.URL, "", "")
	m := config.MavenGav{
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

func TestDownloadFromNexus3Server(t *testing.T) {
	b, err := createZipFile()
	if err != nil {
		t.Error(err.Error())
	}
	zipFileName := "leveransepakke.zip"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(b.Bytes())))
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Server", "Nexus/3.18.1-01 (PRO)")
		w.Write(b.Bytes())
	}))
	defer ts.Close()

	n := NewNexusDownloader(ts.URL, "", "")
	m := config.MavenGav{
		ArtifactId: "openshift-resource-monitor",
		GroupId:    "ske.fellesplattform.monitor",
		Version:    "1.1.4",
		Type:       "zip",
		Classifier: "leveransepakke",
	}

	r, err := n.DownloadArtifact(&m)
	if err != nil {
		t.Error(err.Error())
	}

	if strings.Contains(r.Path, zipFileName) == false {
		t.Error(
			"expected", zipFileName,
			"got", r)
	}
}

func TestNewLocalDownloader(t *testing.T) {
	d := NewBinaryDownloader("test")
	m := config.MavenGav{
		ArtifactId: "dontexist",
		GroupId:    "ske",
		Version:    "develop-SNAPSHOT",
	}
	l, err := d.DownloadArtifact(&m)

	expected := "test"
	if l.Path != expected {
		t.Errorf("Expexted %s, was %s", expected, l)
	}
	if err == nil {
		t.Error("Error expected on non existing file")
	}
}

func TestGetSnapshotTimestampVersion(t *testing.T) {
	expectedVersion := "SNAPSHOT-feature_baz-20170701.103015-1"

	gav := config.MavenGav{
		ArtifactId: "myapp",
		GroupId:    "ske.foo.bar",
		Version:    "feature-baz-SNAPSHOT",
		Classifier: config.Leveransepakke,
		Type:       config.ZipPackaging,
	}

	deliverable := Deliverable{
		Path: "/tmp/package917376626/myapp-feature_baz-20170701.103015-1-Leveransepakke.zip",
	}

	actualVersion := GetSnapshotTimestampVersion(gav, deliverable)

	if actualVersion != expectedVersion {
		t.Errorf("Expexted version %s, actual version was %s", expectedVersion, actualVersion)
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

func TestMavenDownloaderOnSnapshot(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if "/repository/maven-intern/no/skatteetaten/fastsetting/avgift/motorvogn/part3Klient/PTMAF_3298_nullpointer_i_kjregister_ved_henting_av_adr-SNAPSHOT/maven-metadata.xml" == r.RequestURI {

			data, err := os.ReadFile("testdata/manifest.xml")
			assert.NoError(t, err)

			w.WriteHeader(200)
			w.Write(data)
			return
		} else if "/repository/maven-intern/no/skatteetaten/fastsetting/avgift/motorvogn/part3Klient/PTMAF_3298_nullpointer_i_kjregister_ved_henting_av_adr-SNAPSHOT/part3Klient-PTMAF_3298_nullpointer_i_kjregister_ved_henting_av_adr-20220112.075052-2-Leveransepakke.zip" == r.RequestURI {
			data, err := createZipFile()
			assert.NoError(t, err)
			w.WriteHeader(200)
			w.Write(data.Bytes())
			return
		} else {
			t.Errorf("Unexpected call %s", r.RequestURI)
		}
	}))
	defer srv.Close()

	mavenDownloader := NewMavenDownloader(srv.URL, "username", "password")

	maven := config.MavenGav{
		ArtifactId: "part3Klient",
		GroupId:    "no.skatteetaten.fastsetting.avgift.motorvogn",
		Version:    "PTMAF_3298_nullpointer_i_kjregister_ved_henting_av_adr-SNAPSHOT",
		Classifier: "Leveransepakke",
		Type:       "zip",
	}

	deliverable, err := mavenDownloader.DownloadArtifact(&maven)
	assert.NoError(t, err)
	fmt.Println(deliverable.Path)

}

func TestMavenDownloaderOnRelease(t *testing.T) {
	//TODO: Implement me
}
