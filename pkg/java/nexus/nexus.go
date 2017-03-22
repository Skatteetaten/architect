package nexus

import (
	"fmt"
	"github.com/docker/docker/pkg/homedir"
	"github.com/skatteetaten/architect/pkg/config"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type Downloader interface {
	DownloadArtifact(c *config.MavenGav) (string, error)
}

type Nexus struct {
	baseUrl string
}

type LocalRepo struct {
}

func NewNexusDownloader(baseUrl string) Downloader {
	return &Nexus{
		baseUrl: baseUrl,
	}
}

func NewLocalDownloader() Downloader {
	return &LocalRepo{}
}

func (n *LocalRepo) DownloadArtifact(c *config.MavenGav) (string, error) {
	home := homedir.Get()
	replacer := strings.NewReplacer(".", "/")
	path := home + "/.m2/repository/" + replacer.Replace(c.GroupId) +
		"/" + c.ArtifactId + "/" + c.Version + "/" + c.ArtifactId + "-" + c.Version + "-Leveransepakke.zip"
	if _, err := os.Stat(path); err != nil {
		return path, err
	}
	return path, nil
}

func (n *Nexus) DownloadArtifact(c *config.MavenGav) (string, error) {
	resourceUrl, err := n.createURL(c)
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

func (m *Nexus) createURL(n *config.MavenGav) (string, error) {
	tmpUrl, err := url.Parse(m.baseUrl)
	if err != nil {
		return "", err
	}
	query := tmpUrl.Query()
	query.Set("g", n.GroupId)
	query.Set("a", n.ArtifactId)
	query.Set("v", n.Version)
	query.Set("e", "zip")
	query.Set("c", "Leveransepakke")
	query.Set("r", "public-with-staging")
	tmpUrl.RawQuery = query.Encode()
	return tmpUrl.String(), nil
}
