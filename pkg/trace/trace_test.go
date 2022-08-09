package trace

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateAppReg(t *testing.T) {

	srv := FakeEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		deployableImage := &DeployableImage{}
		_ = json.NewDecoder(r.Body).Decode(deployableImage)
		assert.Equal(t, "deployableImage", deployableImage.Type)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/v1/image", r.RequestURI)
	})

	defer srv.Close()

	client2 := NewTraceClient(srv.URL)

	tags := make(map[string]string)
	tags["a"] = "c"
	client2.AddImageMetadata(DeployableImage{
		Type:       "deployableImage",
		Digest:     "manifest.Digest",
		Name:       "buildConfig.DockerRepository",
		AppVersion: "1.2.3",
	})

}

func FakeEndpoint(t *testing.T, endpoint func(w http.ResponseWriter, r *http.Request)) *httptest.Server {

	srv := httptest.NewServer(http.HandlerFunc(endpoint))

	return srv
}
