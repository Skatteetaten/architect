package nexus

import (
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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

//Downloader interface
type Downloader interface {
	DownloadArtifact(c *config.MavenGav) (Deliverable, error)
}

//NexusDownloader struct
type NexusDownloader struct {
	baseURL  string
	username string
	password string
}

//BinaryDownloader struct
type BinaryDownloader struct {
	Path string
}

//Deliverable struct
type Deliverable struct {
	Path string
	SHA1 string
}

//Dependency struct
type Dependency struct {
	Name string
	SHA1 string
	Size int64
}

//NewNexusDownloader of type Downloader
func NewNexusDownloader(baseURL string, username string, password string) Downloader {
	return &NexusDownloader{
		baseURL:  baseURL,
		username: username,
		password: password,
	}
}

//NewBinaryLoader new BinaryDownloader of type Downloader
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
	hash, _ := hashFileSHA1(deliverable.Path)
	deliverable.SHA1 = hash

	return deliverable, nil
}

//DownloadArtifact downloads artifact with given GAV
func (n *NexusDownloader) DownloadArtifact(c *config.MavenGav) (Deliverable, error) {

	deliverable := Deliverable{}

	// Detect API version
	useNexus3 := false
	resp, err := http.Get(n.baseURL)
	if err != nil {
		logrus.Fatalf("Failed response from %s, error: %s", n.baseURL, err)
	} else {
		useNexus3, _ = regexp.MatchString(`^Nexus/3\..*$`, resp.Header.Get("Server"))
		logrus.Infof("Use nexus 3: %t", useNexus3)
	}
	defer resp.Body.Close()

	resourceURL, err := n.resourceURL(c, useNexus3)
	if err != nil {
		return deliverable, errors.Wrap(err, "Failed to create resource url")
	}
	logrus.Infof("Downloading artifact from %s", resourceURL)

	httpClient := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	var location = ""
	var httpResponse *http.Response
	var nextURL = resourceURL
	for { // repeat until no 302 redirect
		req, err := http.NewRequest("GET", nextURL, nil)
		if err != nil {
			return deliverable, errors.Wrapf(err, "Failed to create request for Nexus url %s", resourceURL)
		}
		if n.username != "" && n.password != "" {
			req.SetBasicAuth(n.username, n.password)
		}
		httpResponse, err = httpClient.Do(req)
		if err != nil {
			return deliverable, errors.Wrapf(err, "Error when downloading artifact from %s", resourceURL)
		}
		if httpResponse.StatusCode == http.StatusFound {
			location = httpResponse.Header.Get("Location")
			logrus.Infof("Got redirect to location: %s", location)
			nextURL = location
			_ = httpResponse.Body.Close()
		} else if httpResponse.StatusCode == http.StatusOK {
			break
		} else {
			return deliverable, errors.Errorf("Unhandled status code %s when downloading from %s",
				httpResponse.Status, httpResponse.Request.URL)
		}
	}
	defer httpResponse.Body.Close()

	if httpResponse.StatusCode != http.StatusOK {
		return deliverable, errors.Errorf("Could not download artifact (Make sure you have deployed it!)"+
			". Status code %s , Location %s", httpResponse.Status, httpResponse.Request.URL)
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

	hash, _ := hashFileSHA1(deliverable.Path)
	deliverable.SHA1 = hash

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

func (n *NexusDownloader) resourceURL(cfg *config.MavenGav, useNexus3 bool) (string, error) {
	if useNexus3 {
		resourceURL, err := n.createNexus3URL(cfg)
		if err != nil {
			return resourceURL, errors.Wrapf(err, "Failed to create Nexus 3 url for GAV %+v", cfg)
		}
		return resourceURL, nil
	}
	resourceURL, err := n.createURL(cfg)
	if err != nil {
		return resourceURL, errors.Wrapf(err, "Failed to create Nexus url for GAV %+v", cfg)
	}
	return resourceURL, nil

}

func (n *NexusDownloader) createURL(gav *config.MavenGav) (string, error) {
	tmpURL, err := url.Parse(n.baseURL)
	if err != nil {
		return "", errors.Wrapf(err, "Failed to parse url")
	}
	query := tmpURL.Query()
	query.Set("g", gav.GroupId)
	query.Set("a", gav.ArtifactId)
	query.Set("v", gav.Version)
	query.Set("e", string(gav.Type))
	query.Set("c", string(gav.Classifier))
	query.Set("r", "public-with-staging")
	tmpURL.RawQuery = query.Encode()
	return tmpURL.String(), nil
}

func (n *NexusDownloader) createNexus3URL(gav *config.MavenGav) (string, error) {
	tmpURL, err := url.Parse(n.baseURL + "/service/rest/v1/search/assets/download")
	if err != nil {
		return "", errors.Wrapf(err, "Failed to parse url")
	}
	query := tmpURL.Query()
	query.Set("sort", "version")
	query.Set("repository", "maven-intern")
	query.Set("maven.groupId", gav.GroupId)
	query.Set("maven.artifactId", gav.ArtifactId)
	query.Set("maven.extension", string(gav.Type))
	query.Set("maven.classifier", string(gav.Classifier))
	if gav.IsSnapshot() {
		query.Set("maven.baseVersion", gav.Version)
	} else {
		query.Set("version", gav.Version)
	}
	tmpURL.RawQuery = query.Encode()
	return tmpURL.String(), nil
}

func (n *NexusDownloader) fileName(cfg *config.MavenGav, contentDisposition string, location string) (string, error) {
	if len(contentDisposition) > 0 {
		_, params, err := mime.ParseMediaType(contentDisposition)
		if err != nil {
			return "", errors.Wrap(err, "Failed to parse content-disposition")
		}
		fName := params["filename"]
		if fName != "" {
			return fName, nil
		}
	}

	if location != "" {
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
