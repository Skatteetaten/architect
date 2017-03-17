package nexus

import (
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
)

type Downloader interface {
	DownloadArtifact() (string, error)
}

type Nexus struct {
	BaseUrl    string
	ArtifactId string
	GroupId    string
	Version    string
	Type       string
}

func (n* Nexus) DownloadArtifact() (string, error) {
	resourceUrl, err := createURL(n)
	if err != nil {
		return "", err
	}

	httpResponse, err := http.Get(resourceUrl)
	if err != nil {
		return "", err
	}
	defer httpResponse.Body.Close()

	contentDisposition := httpResponse.Header.Get("content-disposition")

	if len(contentDisposition) <= 0 {
		return "", fmt.Errorf("No content-disposition in response header")
	}

	_, params, err := mime.ParseMediaType(contentDisposition)
	if err != nil {
		return "", err
	}
	dir, err := ioutil.TempDir("", "package")
	if err != nil {
		return "", err
	}
	fileName := filepath.Join(dir, params["filename"])

	fileCreated, err := os.Create(fileName)
	if err != nil {
		return "", err
	}
	defer fileCreated.Close()

	_, err = io.Copy(fileCreated, httpResponse.Body)
	if err != nil {
		return "", err
	}

	return fileName, nil
}

func createURL(n* Nexus) (string, error) {
	tmpUrl, err := url.Parse(n.BaseUrl)
	if err != nil {
		return "", err
	}
	query := tmpUrl.Query()
	query.Set("g", n.GroupId)
	query.Set("a", n.ArtifactId)
	query.Set("v", n.Version)
	query.Set("e", n.Type)
	query.Set("c", "Leveransepakke")
	query.Set("r", "public-with-staging")
	tmpUrl.RawQuery = query.Encode()
	return tmpUrl.String(), nil
}
