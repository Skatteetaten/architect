package trace

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/sirupsen/logrus"
	"github.com/skatteetaten/architect/v2/pkg/config"
	"github.com/skatteetaten/architect/v2/pkg/config/runtime"
	"github.com/skatteetaten/architect/v2/pkg/docker"
	"net/http"
	"time"
)

func NewTracer(sporingsURL string, context string) *Tracer {
	return &Tracer{
		url:     sporingsURL,
		context: context,
		enabled: sporingsURL != "" && context != "",
	}
}

type Tracer struct {
	url     string
	context string
	enabled bool
}

func (t *Tracer) AddImageMetadata(data interface{}) {
	if t.enabled {
		ctx := context.Background()
		timeoutIn := time.Now().Add(5 * time.Second)
		ctx, cancelFunc := context.WithDeadline(ctx, timeoutIn)
		defer cancelFunc()

		d, err := json.Marshal(data)
		if err != nil {
			logrus.Warnf("Unable to unmarshal image metadata. Got error %s", err)
			return
		}
		t.send(ctx, string(d))
	}
}

func (t *Tracer) send(ctx context.Context, jsonStr string) {
	uri := t.url + "/api/v1/trace/" + t.context

	if t.enabled {
		req, err := http.NewRequestWithContext(ctx, "POST", uri, bytes.NewBuffer([]byte(jsonStr)))
		if err != nil {
			logrus.Warnf("Unable to create request: %s", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			logrus.Warnf("Request failed: %s", err)
			return
		}
		defer resp.Body.Close()
	}
}

func (t *Tracer) AddBaseImageMetadata(application config.ApplicationSpec, imageInfo *runtime.ImageInfo, containerConfig *docker.ContainerConfig) {

	payload := BaseImage{
		Type:        "baseImage",
		Name:        application.BaseImageSpec.BaseImage,
		Version:     application.BaseImageSpec.BaseVersion,
		Digest:      imageInfo.Digest,
		ImageConfig: containerConfig,
	}
	logrus.Debugf("Pushing trace data %v", payload)
	t.AddImageMetadata(payload)

}
