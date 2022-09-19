package sporingslogger

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/anchore/syft/syft"
	"github.com/anchore/syft/syft/pkg/cataloger"
	"github.com/anchore/syft/syft/sbom"
	"github.com/anchore/syft/syft/source"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/skatteetaten/architect/v2/pkg/config"
	"github.com/skatteetaten/architect/v2/pkg/config/runtime"
	"github.com/skatteetaten/architect/v2/pkg/docker"
	"github.com/skatteetaten/architect/v2/pkg/sporingslogger/sbomFormat"
	"net/http"
	"net/http/httputil"
	"time"
)

// Sporingslogger interface
type Sporingslogger interface {
	SendImageMetadata(data interface{}) error
	SendBaseImageMetadata(application config.ApplicationSpec, imageInfo *runtime.ImageInfo, containerConfig *docker.ContainerConfig)
	ScanImage(buildFolder string) ([]Dependency, error)
}

// NewClient create new Sporingslogger client
func NewClient(sporingURL string) Sporingslogger {
	return &sporingsloggerClient{
		url:     sporingURL,
		enabled: sporingURL != "",
	}
}

type sporingsloggerClient struct {
	url     string
	enabled bool
}

// SendImageMetadata send image metadata to sporingslogger
func (sporingsloggerClient *sporingsloggerClient) SendImageMetadata(data interface{}) error {
	ctx := context.Background()
	timeoutIn := time.Now().Add(5 * time.Second)
	ctx, cancelFunc := context.WithDeadline(ctx, timeoutIn)
	defer cancelFunc()

	d, err := json.Marshal(data)
	if err != nil {
		return errors.Wrapf(err, "Unable to unmarshal image metadata")
	}
	return sporingsloggerClient.send(ctx, string(d))

	return nil
}

func (sporingsloggerClient *sporingsloggerClient) send(ctx context.Context, jsonStr string) error {
	uri := sporingsloggerClient.url + "/api/v1/image"

	req, err := http.NewRequestWithContext(ctx, "POST", uri, bytes.NewBuffer([]byte(jsonStr)))
	if err != nil {
		logrus.Warnf("Unable to create request: %s", err)
		return errors.Wrapf(err, "Unable to create request")
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	defer resp.Body.Close()

	if err != nil {
		logrus.Warnf("Request failed: %s", err)
		return errors.Wrapf(err, "Request failed")
	}
	if resp.StatusCode >= 300 {
		errorBody, _ := httputil.DumpResponse(resp, true)
		logrus.Warnf("Request failed, error from Sporingslogger:  %s", errorBody)
		return errors.Wrapf(err, "Request failed %d", resp.StatusCode)
	}
	return nil
}

// SendBaseImageMetadata send baseimage metadata to sporingslogger
func (sporingsloggerClient *sporingsloggerClient) SendBaseImageMetadata(application config.ApplicationSpec, imageInfo *runtime.ImageInfo, containerConfig *docker.ContainerConfig) {
	payload := BaseImage{
		Type:        "baseImage",
		Name:        application.BaseImageSpec.BaseImage,
		Version:     application.BaseImageSpec.BaseVersion,
		Digest:      imageInfo.Digest,
		ImageConfig: containerConfig,
	}
	logrus.Debugf("Pushing sporingslogger data %v", payload)
	sporingsloggerClient.SendImageMetadata(payload)

}

// use syft to discover packages + distro only
func (sporingsloggerClient *sporingsloggerClient) ScanImage(buildFolder string) ([]Dependency, error) {

	imageUrl := "dir:" + buildFolder
	input, err := source.ParseInput(imageUrl, "", false)
	if err != nil {
		return nil, err
	}

	src, cleanup, err := source.New(*input, nil, nil)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	catalog, _, release, err := syft.CatalogPackages(src, cataloger.Config{
		Search: cataloger.SearchConfig{
			Scope: source.SquashedScope,
		},
	})
	if err != nil {
		return nil, err
	}

	resultSbom := sbom.SBOM{
		Artifacts: sbom.Artifacts{
			PackageCatalog:    catalog,
			LinuxDistribution: release,
		},
		Source: src.Metadata,
	}
	data, err := syft.Encode(resultSbom, sbomFormat.Format())
	if err != nil {
		return nil, errors.Wrapf(err, "syft.Encode failed")
	}
	var dependencies []Dependency
	sbomReader := bytes.NewReader(data)

	err = json.NewDecoder(sbomReader).Decode(&dependencies)
	if err != nil {
		return nil, errors.Wrapf(err, "Decode failed %v", dependencies)
	}
	return dependencies, nil
}
