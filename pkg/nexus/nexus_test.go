package nexus

import (
	"archive/zip"
	"bytes"
	"github.com/skatteetaten/architect/pkg/config"
	"github.com/stretchr/testify/assert"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

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
		if "/repository/maven-intern/no/skatteetaten/aurora/architect/test_build-SNAPSHOT/maven-metadata.xml" == r.RequestURI {

			data, err := os.ReadFile("testdata/manifest.xml")
			assert.NoError(t, err)

			w.WriteHeader(200)
			w.Write(data)
			return
		} else if "/repository/maven-intern/no/skatteetaten/aurora/architect/test_build-SNAPSHOT/architect-test_build-20220112.075052-2-Leveransepakke.zip" == r.RequestURI {
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
		ArtifactId: "architect",
		GroupId:    "no.skatteetaten.aurora",
		Version:    "test_build-SNAPSHOT",
		Classifier: "Leveransepakke",
		Type:       "zip",
	}

	_, err := mavenDownloader.DownloadArtifact(&maven)
	assert.NoError(t, err)
}

func TestMavenDownloaderOnRelease(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if "/repository/maven-intern/no/skatteetaten/aurora/architect/1.0.0/architect-1.0.0-Leveransepakke.zip" == r.RequestURI {
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
		ArtifactId: "architect",
		GroupId:    "no.skatteetaten.aurora",
		Version:    "1.0.0",
		Classifier: "Leveransepakke",
		Type:       "zip",
	}

	_, err := mavenDownloader.DownloadArtifact(&maven)
	assert.NoError(t, err)
}
