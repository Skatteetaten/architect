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
	"regexp"
	"strings"
)

type Downloader interface {
	DownloadArtifact(c *config.MavenGav, ns *config.NexusAccess) (Deliverable, error)
}

type NexusDownloader struct {
	baseUrl string
}

type BinaryDownloader struct {
	Path string
}

type Deliverable struct {
	Path string
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

func (n *BinaryDownloader) DownloadArtifact(c *config.MavenGav, na *config.NexusAccess) (Deliverable, error) {
	deliverable := Deliverable{
		Path: n.Path,
	}
	if _, err := os.Stat(n.Path); err != nil {
		return deliverable, errors.Wrapf(err, "Failed to stat local artifact %s", n.Path)
	}
	return deliverable, nil
}

func (n *NexusDownloader) DownloadArtifact(c *config.MavenGav, na *config.NexusAccess) (Deliverable, error) {

	deliverable := Deliverable{}

	// Detect API version
	useNexus3 := false
	resp, err := http.Get(n.baseUrl)
	if err != nil {
		logrus.Fatalf("Failed response from %s, error: %s", n.baseUrl, err)
	} else {
		useNexus3, _ = regexp.MatchString(`^Nexus/3\..*$`, resp.Header.Get("Server"))
		logrus.Infof("Use nexus 3: %t", useNexus3)
	}
	defer resp.Body.Close()

	resourceUrl, err := n.resourceURL(c, useNexus3)
	if err != nil {
		return deliverable, errors.Wrap(err, "Failed to create resource url")
	}
	logrus.Infof("Downloading artifact from %s", resourceUrl)

	httpClient := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	var location = ""
	var httpResponse *http.Response
	var nextURL = resourceUrl
	for { // repeat until no 302 redirect
		req, err := http.NewRequest("GET", nextURL, nil)
		if err != nil {
			return deliverable, errors.Wrapf(err, "Failed to create request for Nexus url %s", resourceUrl)
		}
		if na != nil && na.Username != "" && na.Password != "" {
			req.SetBasicAuth(na.Username, na.Password)
		}
		httpResponse, err = httpClient.Do(req)
		if httpResponse.StatusCode == http.StatusFound {
			location = httpResponse.Header.Get("Location")
			logrus.Infof("Got redirect to location: %s", location)
			nextURL = location
			httpResponse.Body.Close()
		} else if err != nil {
			return deliverable, errors.Wrapf(err, "Failed to get artifact from Nexus %s", resourceUrl)
		} else {
			break
		}
	}
	defer httpResponse.Body.Close()

	if httpResponse.StatusCode != http.StatusOK {
		return deliverable, errors.Errorf("Could not download artifact (Make sure you have deployed it!)"+
			". Status code %s ", httpResponse.Status)
	}

	fileName, err := n.fileName(c, httpResponse.Header.Get("content-disposition"), location)
	if err != nil {
		return deliverable, errors.Wrapf(err, "Could not create filename for temporary file")
	}

	dir, err := ioutil.TempDir("", "package")
	if err != nil {
		return deliverable, errors.Wrap(err, "Failed to create directory for artifact")
	}
	filePath := filepath.Join(dir, fileName)

	fileCreated, err := os.Create(filePath)
	if err != nil {
		return deliverable, errors.Wrap(err, "Failed to create artifact file")
	}
	defer fileCreated.Close()

	_, err = io.Copy(fileCreated, httpResponse.Body)
	if err != nil {
		return deliverable, errors.Wrap(err, "Failed to write to artifact file")
	}
	deliverable.Path = filePath
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

func (m *NexusDownloader) resourceURL(cfg *config.MavenGav, useNexus3 bool) (string, error) {
	if useNexus3 {
		resourceUrl, err := m.createNexus3URL(cfg)
		if err != nil {
			return resourceUrl, errors.Wrapf(err, "Failed to create Nexus 3 url for GAV %+v", cfg)
		}
		return resourceUrl, nil
	} else {
		resourceUrl, err := m.createURL(cfg)
		if err != nil {
			return resourceUrl, errors.Wrapf(err, "Failed to create Nexus url for GAV %+v", cfg)
		}
		return resourceUrl, nil
	}
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

func (m *NexusDownloader) createNexus3URL(n *config.MavenGav) (string, error) {
	tmpUrl, err := url.Parse(m.baseUrl + "/service/rest/v1/search/assets/download")
	if err != nil {
		return "", errors.Wrapf(err, "Failed to parse url")
	}
	query := tmpUrl.Query()
	query.Set("sort", "version")
	query.Set("repository", "maven-intern")
	query.Set("maven.groupId", n.GroupId)
	query.Set("maven.artifactId", n.ArtifactId)
	query.Set("maven.extension", string(n.Type))
	query.Set("maven.classifier", string(n.Classifier))
	if n.IsSnapshot() {
		query.Set("maven.baseVersion", n.Version)
	} else {
		query.Set("version", n.Version)
	}
	tmpUrl.RawQuery = query.Encode()
	return tmpUrl.String(), nil
}

func (m *NexusDownloader) fileName(cfg *config.MavenGav, contentDisposition string, location string) (string, error) {
	if len(contentDisposition) > 0 {
		_, params, err := mime.ParseMediaType(contentDisposition)
		if err != nil {
			return "", errors.Wrap(err, "Failed to parse content-disposition")
		}
		return params["filename"], nil
	} else if location != "" {
		_, fileName := filepath.Split(location)
		return fileName, nil
	} else {
		// No content-disposition or location - using alternate filename composition
		if string(cfg.Classifier) == "" {
			return "", errors.Errorf("Missing maven Classifier")
		}
		if string(cfg.Type) == "" {
			return "", errors.Errorf("Missing maven Type")
		}
		logrus.Warn("Using generic filename for temp file")
		return string(cfg.Classifier) + "." + string(cfg.Type), nil
	}
}
