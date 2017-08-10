package nexus

import (
	"github.com/docker/docker/pkg/homedir"
	"github.com/pkg/errors"
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
	DownloadArtifact(c *config.JavaApplication) (*Deliverable, error)
}

type Nexus struct {
	baseUrl string
}

type LocalRepo struct {
}

type Deliverable struct {
	Path string
}

func NewNexusDownloader(baseUrl string) Downloader {
	return &Nexus{
		baseUrl: baseUrl,
	}
}

func NewLocalDownloader() Downloader {
	return &LocalRepo{}
}

func (n *LocalRepo) DownloadArtifact(c *config.JavaApplication) (*Deliverable, error) {
	home := homedir.Get()
	replacer := strings.NewReplacer(".", "/")
	path := home + "/.m2/repository/" + replacer.Replace(c.GroupId) +
		"/" + c.ArtifactId + "/" + c.Version + "/" + c.ArtifactId + "-" + c.Version + "-Leveransepakke.zip"
	if _, err := os.Stat(path); err != nil {
		return &Deliverable{path}, errors.Wrapf(err, "Failed to stat local artifact %s", path)
	}
	return &Deliverable{path}, nil
}

func (n *Nexus) DownloadArtifact(c *config.JavaApplication) (*Deliverable, error) {
	resourceUrl, err := n.createURL(c)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create Nexus url for GAV %+v", c)
	}

	httpResponse, err := http.Get(resourceUrl)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get artifact from Nexus %s", resourceUrl)
	}
	defer httpResponse.Body.Close()

	contentDisposition := httpResponse.Header.Get("content-disposition")

	if len(contentDisposition) <= 0 {
		return nil, errors.Errorf("No content-disposition in response header")
	}

	_, params, err := mime.ParseMediaType(contentDisposition)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to parse content-disposition")
	}
	dir, err := ioutil.TempDir("", "package")
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create directory for artifact")
	}
	fileName := filepath.Join(dir, params["filename"])

	fileCreated, err := os.Create(fileName)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create artifact file")
	}
	defer fileCreated.Close()

	_, err = io.Copy(fileCreated, httpResponse.Body)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to write to artifact file")
	}

	return &Deliverable{fileName}, nil
}

func (m *Nexus) createURL(n *config.JavaApplication) (string, error) {
	tmpUrl, err := url.Parse(m.baseUrl)
	if err != nil {
		return "", errors.Wrapf(err, "Failed to parse url")
	}
	query := tmpUrl.Query()
	query.Set("g", n.GroupId)
	query.Set("a", n.ArtifactId)
	query.Set("v", n.Version)
	query.Set("e", "zip")
	query.Set("c", n.Classifier)
	query.Set("r", "public-with-staging")
	tmpUrl.RawQuery = query.Encode()
	return tmpUrl.String(), nil
}
