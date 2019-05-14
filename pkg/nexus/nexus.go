package nexus

import (
	"github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	"github.com/skatteetaten/architect/pkg/config"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type Downloader interface {
	DownloadArtifact(c *config.MavenGav) (Deliverable, error)
}

type NexusDownloader struct {
	baseUrl string
}

type PathAwareNexusDownloader struct {
	baseUrl, path string
}

type BinaryDownloader struct {
	Path string
}

type Deliverable struct {
	Path string
}

func NewPathAwareNexusDownloader(baseUrl, path string) Downloader {
	return &PathAwareNexusDownloader{
		baseUrl: baseUrl,
		path:    path,
	}
}

func NewNexusDownloader(baseUrl string) Downloader {
	return &NexusDownloader{
		baseUrl: baseUrl,
	}
}

func NewBinaryDownloader(path string) Downloader {
	return &BinaryDownloader{
		Path: path,
	}
}

func (n *BinaryDownloader) DownloadArtifact(c *config.MavenGav) (Deliverable, error) {
	deliverable := Deliverable{
		Path: n.Path,
	}
	if _, err := os.Stat(n.Path); err != nil {
		return deliverable, errors.Wrapf(err, "Failed to stat local artifact %s", n.Path)
	}
	return deliverable, nil
}

func (n *PathAwareNexusDownloader) DownloadArtifact(c *config.MavenGav) (Deliverable, error) {
	resourceUrl, err := n.createURL(c)
	deliverable := Deliverable{}
	if err != nil {
		return deliverable, errors.Wrapf(err, "Failed to create Nexus url for GAV %+v", c)
	}

	logrus.Debugf("Downloading artifact from %s", resourceUrl)
	httpResponse, err := http.Get(resourceUrl)
	if err != nil {
		return deliverable, errors.Wrapf(err, "Failed to get artifact from Nexus %s", resourceUrl)
	}

	defer httpResponse.Body.Close()

	if httpResponse.StatusCode != http.StatusOK {
		return deliverable, errors.Errorf("Could not download artifact (Make sure you have deployed it!)"+
			". Status code %s ", httpResponse.Status)
	}

	contentDisposition := httpResponse.Header.Get("content-disposition")

	if len(contentDisposition) <= 0 {
		return deliverable, errors.Errorf("No content-disposition in response header")
	}

	_, params, err := mime.ParseMediaType(contentDisposition)
	if err != nil {
		return deliverable, errors.Wrap(err, "Failed to parse content-disposition")
	}

	fileName := filepath.Join(n.path, params["filename"])
	logrus.Infof("Save downloaded file to location %s", fileName)

	fileCreated, err := os.Create(fileName)
	if err != nil {
		return deliverable, errors.Wrap(err, "Failed to create artifact file")
	}
	defer fileCreated.Close()

	_, err = io.Copy(fileCreated, httpResponse.Body)
	if err != nil {
		return deliverable, errors.Wrap(err, "Failed to write to artifact file")
	}
	deliverable.Path = fileName
	logrus.Debugf("Downloaded artifact to %s", deliverable.Path)
	return deliverable, nil

}

func (n *NexusDownloader) DownloadArtifact(c *config.MavenGav) (Deliverable, error) {
	resourceUrl, err := n.createURL(c)
	deliverable := Deliverable{}
	if err != nil {
		return deliverable, errors.Wrapf(err, "Failed to create Nexus url for GAV %+v", c)
	}

	logrus.Debugf("Downloading artifact from %s", resourceUrl)
	httpResponse, err := http.Get(resourceUrl)
	if err != nil {
		return deliverable, errors.Wrapf(err, "Failed to get artifact from Nexus %s", resourceUrl)
	}
	defer httpResponse.Body.Close()

	if httpResponse.StatusCode != http.StatusOK {
		return deliverable, errors.Errorf("Could not download artifact (Make sure you have deployed it!)"+
			". Status code %s ", httpResponse.Status)
	}

	contentDisposition := httpResponse.Header.Get("content-disposition")

	if len(contentDisposition) <= 0 {
		return deliverable, errors.Errorf("No content-disposition in response header")
	}

	_, params, err := mime.ParseMediaType(contentDisposition)
	if err != nil {
		return deliverable, errors.Wrap(err, "Failed to parse content-disposition")
	}
	dir, err := ioutil.TempDir("", "package")
	if err != nil {
		return deliverable, errors.Wrap(err, "Failed to create directory for artifact")
	}
	fileName := filepath.Join(dir, params["filename"])

	fileCreated, err := os.Create(fileName)
	if err != nil {
		return deliverable, errors.Wrap(err, "Failed to create artifact file")
	}
	defer fileCreated.Close()

	_, err = io.Copy(fileCreated, httpResponse.Body)
	if err != nil {
		return deliverable, errors.Wrap(err, "Failed to write to artifact file")
	}
	deliverable.Path = fileName
	logrus.Debugf("Downloaded artifact to %s", deliverable.Path)
	return deliverable, nil
}

/*
  Create app version. If not snapshot build, then return version from GAV.
  Otherwise, create new snapshot version based on deliverable.
*/
func GetSnapshotTimestampVersion(gav config.MavenGav, deliverable Deliverable) string {
	if gav.IsSnapshot() {
		replacer := strings.NewReplacer(gav.ArtifactId+"-", "", "-"+string(gav.Classifier)+"."+string(gav.Type), "")
		version := "SNAPSHOT-" + replacer.Replace(path.Base(deliverable.Path))
		return version
	}
	return gav.Version
}

func (m *NexusDownloader) createURL(n *config.MavenGav) (string, error) {
	tmpUrl, err := url.Parse(m.baseUrl)
	if err != nil {
		return "", errors.Wrapf(err, "Failed to parse url")
	}
	query := tmpUrl.Query()
	query.Set("g", n.GroupId)
	query.Set("a", n.ArtifactId)
	query.Set("v", n.Version)
	query.Set("e", string(n.Type))
	query.Set("c", string(n.Classifier))
	query.Set("r", "public-with-staging")
	tmpUrl.RawQuery = query.Encode()
	return tmpUrl.String(), nil
}

func (m *PathAwareNexusDownloader) createURL(n *config.MavenGav) (string, error) {
	tmpUrl, err := url.Parse(m.baseUrl)
	if err != nil {
		return "", errors.Wrapf(err, "Failed to parse url")
	}
	query := tmpUrl.Query()
	query.Set("g", n.GroupId)
	query.Set("a", n.ArtifactId)
	query.Set("v", n.Version)
	query.Set("e", string(n.Type))
	query.Set("c", string(n.Classifier))
	query.Set("r", "public-with-staging")
	tmpUrl.RawQuery = query.Encode()

	logrus.Info("URL: ", tmpUrl.RawQuery)
	return tmpUrl.String(), nil

}
