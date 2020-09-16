package docker

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const repository = "aurora/flange"
const tag = "8"
const digest = "sha256:b6a7c668428ff9347ef5c4f8736e8b7f38696dc6acc74409627d360752017fcc"

func TestGetManifestEnvSchemaV1(t *testing.T) {
	expectedVersion := "1.7.0"

	server, err := startMockRegistryServer("testdata/manifest.json")
	defer server.Close()
	assert.NoError(t, err)

	target := NewRegistryClient(server.URL, server.URL, nil)
	imageInfo, err := target.GetImageInfo(context.Background(), repository, tag)
	assert.NoError(t, err)

	actualVersion := imageInfo.Enviroment["BASE_IMAGE_VERSION"]
	assert.Equal(t, expectedVersion, actualVersion)
}

func TestGetCompleteBaseImageVersionSchemaV1(t *testing.T) {
	expectedVersion := "1.7.0"

	server, err := startMockRegistryServer("testdata/manifest.json")
	defer server.Close()
	assert.NoError(t, err)

	target := NewRegistryClient(server.URL, server.URL, nil)
	imageInfo, err := target.GetImageInfo(context.Background(), repository, tag)
	assert.NoError(t, err)
	assert.Equal(t, expectedVersion, imageInfo.CompleteBaseImageVersion)
}

func TestGetManifestEnvMapSchemaV1(t *testing.T) {
	expectedLength := 14

	server, err := startMockRegistryServer("testdata/manifest.json")
	defer server.Close()
	assert.NoError(t, err)

	target := NewRegistryClient(server.URL, server.URL, nil)
	imageInfo, err := target.GetImageInfo(context.Background(), repository, tag)
	assert.NoError(t, err)

	actualLength := len(imageInfo.Enviroment)
	assert.Equal(t, expectedLength, actualLength)
}

func TestGetManifestEnvSchemaV2(t *testing.T) {
	expectedVersion := "8.152.18"

	server, err := startMockRegistryManifestServer("testdata/aurora_flange_manifest_v2.json", "testdata/aurora_flange_container_image_v1.json")
	defer server.Close()
	assert.NoError(t, err)

	target := NewRegistryClient(server.URL, server.URL, nil)
	imageInfo, err := target.GetImageInfo(context.Background(), repository, tag)
	assert.NoError(t, err)

	actualVersion := imageInfo.Enviroment["BASE_IMAGE_VERSION"]
	assert.Equal(t, expectedVersion, actualVersion)
}

func TestGetCompleteBaseImageVersionSchemaV2(t *testing.T) {
	expectedVersion := "8.152.18"

	server, err := startMockRegistryManifestServer("testdata/aurora_flange_manifest_v2.json", "testdata/aurora_flange_container_image_v1.json")
	defer server.Close()
	assert.NoError(t, err)

	target := NewRegistryClient(server.URL, server.URL, nil)
	imageInfo, err := target.GetImageInfo(context.Background(), repository, tag)
	assert.NoError(t, err)
	assert.Equal(t, expectedVersion, imageInfo.CompleteBaseImageVersion)
}

func TestGetManifestV2(t *testing.T) {
	server, err := startMockRegistryServer("testdata/aurora_flange_manifest_v2.json")
	defer server.Close()
	assert.NoError(t, err)

	target := NewRegistryClient(server.URL, server.URL, nil)
	manifest, err := target.GetManifest(context.Background(), repository, tag)
	assert.NoError(t, err)
	assert.Equal(t, "sha256:b6a7c668428ff9347ef5c4f8736e8b7f38696dc6acc74409627d360752017fcc", manifest.Config.Digest)
}

func TestGetContainerConfig(t *testing.T) {
	server, err := startMockRegistryServer("testdata/aurora_wingnut11_container_config.json")
	defer server.Close()
	assert.NoError(t, err)

	target := NewRegistryClient(server.URL, server.URL, nil)
	config, err := target.GetContainerConfig(context.Background(), repository, tag)
	assert.NoError(t, err)
	assert.Equal(t, "amd64", config.Architecture)
}

func TestLayerExists(t *testing.T) {

	t.Run("Layer exists", func(t *testing.T) {
		ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "HEAD", r.Method)
			assert.Equal(t, "/v2/aurora/flange/blobs/8", r.RequestURI)
			w.WriteHeader(200)
		}))
		ts.StartTLS()
		defer ts.Close()

		target := NewRegistryClient(ts.URL, ts.URL, nil)

		ok, err := target.LayerExists(context.Background(), repository, tag)
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("Layer does not exists", func(t *testing.T) {
		ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "HEAD", r.Method)
			assert.Equal(t, "/v2/aurora/flange/blobs/8", r.RequestURI)
			w.WriteHeader(404)
		}))
		ts.StartTLS()
		defer ts.Close()

		target := NewRegistryClient(ts.URL, ts.URL, nil)

		ok, err := target.LayerExists(context.Background(), repository, tag)
		assert.NoError(t, err)
		assert.False(t, ok)
	})
}

func TestMountLayer(t *testing.T) {

	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v2/test/architect/blobs/uploads/?mount=777&from=aurora/wingnut", r.RequestURI)
		w.WriteHeader(201)
	}))
	ts.StartTLS()
	defer ts.Close()

	target := NewRegistryClient(ts.URL, ts.URL, nil)

	err := target.MountLayer(context.Background(), "aurora/wingnut", "test/architect", "777")
	assert.NoError(t, err)
}

func TestPushManifest(t *testing.T) {
	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PUT", r.Method)
		assert.Equal(t, "/v2/test/architect/manifests/latest", r.RequestURI)
		w.WriteHeader(201)
	}))
	ts.StartTLS()
	defer ts.Close()

	target := NewRegistryClient(ts.URL, ts.URL, nil)

	err := target.PushManifest(context.Background(), "testdata/aurora_flange_manifest_v2.json", "test/architect", "latest")
	assert.NoError(t, err)
}

func TestPushLayer(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/v2/test/architect/blobs/uploads/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v2/test/architect/blobs/uploads/", r.RequestURI)

		w.Header().Set("Location", "/v2/start/transaction")
		w.WriteHeader(202)

	})
	mux.HandleFunc("/v2/start/transaction", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PATCH", r.Method)
		w.Header().Set("Location", "/v2/your/upload/location")
		w.WriteHeader(202)
	})

	mux.HandleFunc("/v2/your/upload/location", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PUT", r.Method)
		assert.Equal(t, "/v2/your/upload/location?digest=666", r.RequestURI)
		assert.Equal(t, "application/octet-stream", r.Header.Get("Content-Type"))
		w.WriteHeader(201)
	})

	ts := httptest.NewUnstartedServer(mux)
	ts.StartTLS()
	defer ts.Close()

	target := NewRegistryClient(ts.URL, ts.URL, nil)

	err := target.PushLayer(context.Background(), "testdata/app-layer.tar.gz", "test/architect", "666")
	assert.NoError(t, err)

}

func TestGetManifestEnvMapSchemaV2(t *testing.T) {
	expectedLength := 13

	server, err := startMockRegistryManifestServer("testdata/aurora_flange_manifest_v2.json", "testdata/aurora_flange_container_image_v1.json")
	defer server.Close()
	assert.NoError(t, err)

	target := NewRegistryClient(server.URL, server.URL, nil)
	imageInfo, err := target.GetImageInfo(context.Background(), repository, tag)
	assert.NoError(t, err)

	actualLength := len(imageInfo.Enviroment)
	assert.Equal(t, expectedLength, actualLength)
}

func TestGetTags(t *testing.T) {
	server, err := startMockRegistryServer("testdata/tags.list.json")
	defer server.Close()
	assert.NoError(t, err)

	expectedTags := []string{"latest", "develop-SNAPSHOT",
		"develop-SNAPSHOT-9be2b9ca43a024415947a6c262e183406dbb090b",
		"2.0.0", "1.3.0", "1.2.1", "1.1.2", "1.1", "1.2", "1.3", "2.0", "2", "1"}

	target := NewRegistryClient(server.URL, server.URL, nil)
	tags, err := target.GetTags(context.Background(), "aurora/oracle8")
	assert.NoError(t, err)
	verifyTagListContent(tags.Tags, expectedTags, t)
}

func TestGetTagsWithMeta(t *testing.T) {
	server, err := startMockRegistryServer("testdata/tags.list.meta.json")
	defer server.Close()
	assert.NoError(t, err)

	expectedTags := []string{"latest", "develop-SNAPSHOT",
		"develop-SNAPSHOT-9be2b9ca43a024415947a6c262e183406dbb090b",
		"2.0.0+somemeta2", "1.3.0+somemeta1", "1.2.1", "1.1.2", "1.1", "1.2", "1.3+somemeta1", "2.0+somemeta2", "2+somemeta2", "1+somemeta1"}

	target := NewRegistryClient(server.URL, server.URL, nil)
	tags, err := target.GetTags(context.Background(), "aurora/oracle8")
	assert.NoError(t, err)
	verifyTagListContent(tags.Tags, expectedTags, t)
}

func TestReadingOfEnvStrings(t *testing.T) {
	key, value, err := envKeyValue("JAVA_TOOL_OPTIONS=-Dfile.encoding=UTF-8 -Djava.net.preferIPv4Stack=true")
	assert.NoError(t, err)
	assert.Equal(t, "JAVA_TOOL_OPTIONS", key)
	assert.Equal(t, "-Dfile.encoding=UTF-8 -Djava.net.preferIPv4Stack=true", value)

	key, value, err = envKeyValue("KEY=value")
	assert.NoError(t, err)
	assert.Equal(t, "KEY", key)
	assert.Equal(t, "value", value)

	key, value, err = envKeyValue("KEY=")
	assert.NoError(t, err)
	assert.Equal(t, "KEY", key)
	assert.Equal(t, "", value)

	key, value, err = envKeyValue("KEY")
	assert.Error(t, err)
}

func verifyTagListContent(actualList []string, expectedList []string, t *testing.T) {
	if len(actualList) != len(expectedList) {
		t.Errorf("Expected %d tags, actual is %d", len(expectedList), len(actualList))
	}

	for _, e := range expectedList {
		verifyContainsTag(actualList, e, t)
	}
}

func verifyContainsTag(actual []string, expected string, t *testing.T) {
	if !contains(actual, expected) {
		t.Errorf("Expected tag %s does not exist", expected)
	}
}

func contains(target []string, value string) bool {
	for _, t := range target {
		if t == value {
			return true
		}
	}

	return false
}

func startMockRegistryServer(filename string) (*httptest.Server, error) {
	buf, err := ioutil.ReadFile(filename)

	if err != nil {
		return nil, err
	}

	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(buf)))
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Write(buf)
	}))
	ts.StartTLS()
	return ts, nil
}

func startMockRegistryManifestServer(fileManifest string, fileImageMeta string) (*httptest.Server, error) {
	bufManifest, err := ioutil.ReadFile(fileManifest)

	if err != nil {
		return nil, err
	}

	bufImageMeta, err := ioutil.ReadFile(fileImageMeta)

	if err != nil {
		return nil, err
	}

	bufManifestError, err := ioutil.ReadFile("testdata/aurora_flange_manifest_error.json")

	if err != nil {
		return nil, err
	}

	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var buf []byte
		if strings.Contains(r.Header.Get("Accept"), httpHeaderManifestSchemaV2) {
			buf = bufManifest
		} else if strings.Contains(r.Header.Get("Accept"), httpHeaderContainerImageV1) {
			buf = bufImageMeta
		} else {
			buf = bufManifestError
		}
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(buf)))
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Write(buf)
	}))
	ts.StartTLS()
	return ts, nil
}
