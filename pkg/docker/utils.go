package docker

import (
	"context"
	"github.com/pkg/errors"
	"github.com/skatteetaten/architect/v2/pkg/config"
	"github.com/skatteetaten/architect/v2/pkg/config/runtime"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

// CreateCompleteTagsFromSpecAndTags makes a target image to be used for push.
// The tag format in docker is somewhat confusing. For a description, see
// https://docs.docker.com/engine/reference/commandline/tag/
// Given format a list of tags consisting of "a", "b" and "c", a registry host:5000 and repository "aurora/test",
// this function will return:
// host:5000/aurora/test:a
// host:5000/aurora/test:b
// host:5000/aurora/test:c
func CreateImageNameFromSpecAndTags(tags []string, outputRegistry string, outputRepository string) []string {
	output := make([]string, len(tags))
	for i, t := range tags {
		name := &runtime.DockerImage{
			Registry:   outputRegistry,
			Repository: outputRepository,
			Tag:        ConvertTagToRepositoryTag(t),
		}
		output[i] = name.GetCompleteDockerTagName()
	}
	return output
}

func ConvertTagToRepositoryTag(tag string) string {
	return strings.Replace(tag, "+", "_", -1)
}

func ConvertRepositoryTagToTag(tag string) string {
	return strings.Replace(tag, "_", "+", -1)
}

func ConvertRepositoryTagsToTags(tags []string) []string {
	newTags := make([]string, 0, len(tags))
	for _, tag := range tags {
		newTags = append(newTags, ConvertRepositoryTagToTag(tag))
	}
	return newTags
}

func GetUtcTimestamp() string {
	location, _ := time.LoadLocation("UTC")
	return time.Now().In(location).Format(time.RFC3339)

}

func getHTTPRequest(client *http.Client, ctx context.Context, headers map[string]string, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to create request %s", url)
	}
	for key, value := range headers {
		req.Header.Add(key, value)
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "GET %s failed", url)
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusOK {
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to read requested body for url %s with header %s", url, headers)
		}
		return body, nil
	}

	return nil, errors.Errorf("Unabled to read manifest. From registry: %s", res.Status)
}

func GetPortOrDefault(port string) string {
	if port == "" {
		return "443"
	}
	return port
}

//TODO: HACK: Fix registry certificate. TLS handshake fails with: does not contain any IP SANs
func InsecureOrDefault(config *config.Config) bool {
	if config.BinaryBuild {
		return true
	}
	return false
}
