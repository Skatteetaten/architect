package trace

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/skatteetaten/architect/v2/pkg/config"
	"github.com/skatteetaten/architect/v2/pkg/config/runtime"
	"github.com/skatteetaten/architect/v2/pkg/docker"
	"net/http"
	"time"
)

type Trace interface {
	AddImageMetadata(data interface{}) error
	AddBaseImageMetadata(application config.ApplicationSpec, imageInfo *runtime.ImageInfo, containerConfig *docker.ContainerConfig)
}

func NewTraceClient(sporingURL string) Trace {
	return &TraceClient{
		url:     sporingURL,
		enabled: sporingURL != "",
	}
}

type TraceClient struct {
	url     string
	enabled bool
}

func (traceClient *TraceClient) AddImageMetadata(data interface{}) error {
	ctx := context.Background()
	timeoutIn := time.Now().Add(5 * time.Second)
	ctx, cancelFunc := context.WithDeadline(ctx, timeoutIn)
	defer cancelFunc()

	d, err := json.Marshal(data)
	if err != nil {
		return errors.Wrapf(err, "Unable to unmarshal image metadata")
	}
	return traceClient.send(ctx, string(d))

	return nil
}

func (traceClient *TraceClient) send(ctx context.Context, jsonStr string) error {
	uri := traceClient.url + "/api/v1/image"

	logrus.Info("send info")
	logrus.Info(uri)
	req, err := http.NewRequestWithContext(ctx, "POST", uri, bytes.NewBuffer([]byte(jsonStr)))
	if err != nil {
		logrus.Warnf("Unable to create request: %s", err)
		return errors.Wrapf(err, "Unable to create request")
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logrus.Warnf("Request failed: %s", err)
		return errors.Wrapf(err, "Request failed")
	}
	if resp.StatusCode >= 300 {
		logrus.Warnf("Request failed: %s", err)
		return errors.Wrapf(err, "Request failed %d", resp.StatusCode)
	}

	defer resp.Body.Close()
	return nil
}

func (traceClient *TraceClient) AddBaseImageMetadata(application config.ApplicationSpec, imageInfo *runtime.ImageInfo, containerConfig *docker.ContainerConfig) {
	payload := BaseImage{
		Type:        "baseImage",
		Name:        application.BaseImageSpec.BaseImage,
		Version:     application.BaseImageSpec.BaseVersion,
		Digest:      imageInfo.Digest,
		ImageConfig: containerConfig,
	}
	logrus.Debugf("Pushing trace data %v", payload)
	traceClient.AddImageMetadata(payload)

}
