package sporingslogger

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateAppReg(t *testing.T) {

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deployableImage := &DeployableImage{}
		_ = json.NewDecoder(r.Body).Decode(deployableImage)
		assert.Equal(t, "deployableImage", deployableImage.Type)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/v1/image", r.RequestURI)
	}))

	client2 := NewClient(srv.URL)

	tags := make(map[string]string)
	tags["a"] = "c"
	client2.SendImageMetadata(DeployableImage{
		Type:       "deployableImage",
		Digest:     "manifest.Digest",
		Name:       "buildConfig.DockerRepository",
		AppVersion: "1.2.3",
	})

}
