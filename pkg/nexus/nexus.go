package nexus

import (
	"encoding/xml"
	"fmt"
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

type MavenDownloader struct {
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

func NewMavenDownloader(baseURL string, username string, password string) Downloader {
	return &MavenDownloader{
		baseURL:  baseURL,
		username: username,
		password: password,
	}
}

func (n *MavenDownloader) DownloadArtifact(c *config.MavenGav) (Deliverable, error) {
	deliverable := Deliverable{}

	httpClient := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	mavenManifest := MavenManifest{}
	if c.IsSnapshot() {
		//Handle snapshot
		u, err := url.Parse(n.baseURL)
		if err != nil {
			return deliverable, errors.Wrap(err, "Unable to parse nexus url")
		}
		//Set path
		u.Path = createMavenManifestPath(c)
		if err != nil {
			return deliverable, errors.Wrap(err, "Could no create path from gav")
		}

		logrus.Debugf("Fetch maven manifest from %s", u.String())

		req, err := http.NewRequest("GET", u.String(), nil)
		req.Header.Set("Accept", "application/xml")
		if err != nil {
			return deliverable, errors.Wrapf(err, "Failed to create request for Nexus url %s", u.String())
		}
		if n.username != "" && n.password != "" {
			req.SetBasicAuth(n.username, n.password)
		}
		resp, err := httpClient.Do(req)
		if err != nil {
			return deliverable, errors.Wrapf(err, "Error when downloading manifest from %s", u.String())
		}

		defer resp.Body.Close()

		err = xml.NewDecoder(resp.Body).Decode(&mavenManifest)
		if err != nil {
			return deliverable, errors.Wrap(err, "XML decode failed")
		}
	}

	//Create download url
	u, err := url.Parse(n.baseURL)
	if err != nil {
		return deliverable, errors.Wrap(err, "Unable to parse nexus url")
	}
	//Set path
	u.Path = createDownloadPath(mavenManifest, c)
	if err != nil {
		return deliverable, errors.Wrap(err, "Could no create path from gav")
	}
	logrus.Infof("Downloading artifact from %s", u.String())

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return deliverable, errors.Wrapf(err, "Failed to create request for Nexus url %s", u.String())
	}
	if n.username != "" && n.password != "" {
		req.SetBasicAuth(n.username, n.password)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return deliverable, errors.Wrapf(err, "Error when downloading artifact from %s", u.String())
	}

	location := resp.Header.Get("Location")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return deliverable, errors.Errorf("Could not download artifact (Make sure you have deployed it!)"+
			". Status code %s , Location %s", resp.Status, resp.Request.URL)
	}

	fileName, err := fileName(c, resp.Header.Get("content-disposition"), location)
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

	_, err = io.Copy(fileCreated, resp.Body)
	if err != nil {
		return deliverable, errors.Wrap(err, "Failed to write to artifact file")
	}
	deliverable.Path = filePath
	logrus.Debugf("Downloaded artifact to %s", deliverable.Path)

	hash, _ := hashFileSHA1(deliverable.Path)
	deliverable.SHA1 = hash

	return deliverable, nil

}

type MavenManifest struct {
	Versioning Versioning `xml:"versioning"`
}

type Versioning struct {
	Snapshot Snapshot `xml:"snapshot"`
}

type Snapshot struct {
	Timestamp   string `xml:"timestamp"`
	BuildNumber int    `xml:"buildNumber"`
}

func createMavenManifestPath(c *config.MavenGav) string {
	groupId := strings.ReplaceAll(c.GroupId, ".", "/")
	version := strings.ReplaceAll(c.Version, "-", "_")
	return fmt.Sprintf("/repository/maven/intern/%s/%s/%s/maven-metadata.xml", groupId, c.ArtifactId, version)
}

func createDownloadPath(manifest MavenManifest, c *config.MavenGav) string {
	groupId := strings.ReplaceAll(c.GroupId, ".", "/")
	version := strings.ReplaceAll(c.Version, "-", "_")
	versionWithoutSnapshot := strings.ReplaceAll(c.Version, "SNAPSHOT", "")

	var artifact string
	if c.IsSnapshot() {
		artifact = fmt.Sprintf("%s%s-%d%s", versionWithoutSnapshot, manifest.Versioning.Snapshot.Timestamp, manifest.Versioning.Snapshot.BuildNumber, getClassifierExt(c))
	} else {
		artifact = fmt.Sprintf("%s%s", version, getClassifierExt(c))
	}
	return fmt.Sprintf("/repository/maven/intern/%s/%s/%s/%s", groupId, c.ArtifactId, version, artifact)
}

func getClassifierExt(c *config.MavenGav) string {

	if c.Classifier != "" {
		return fmt.Sprintf("-%s.%s", c.Classifier, c.Type)
	} else {
		return fmt.Sprintf(".%s", c.Type)
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

	fileName, err := fileName(c, httpResponse.Header.Get("content-disposition"), location)
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

func fileName(cfg *config.MavenGav, contentDisposition string, location string) (string, error) {
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
