package nexus

import (
	"encoding/xml"
	"fmt"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/skatteetaten/architect/v2/pkg/config"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// Downloader interface
type Downloader interface {
	DownloadArtifact(c *config.MavenGav) (Deliverable, error)
}

// MavenDownloader configuration
type MavenDownloader struct {
	baseURL  string
	username string
	password string
}

// BinaryDownloader configuration
type BinaryDownloader struct {
	Path string
}

// Deliverable struct
type Deliverable struct {
	Path string
	SHA1 string
}

// Dependency utility struct used for gathering dependency metadata
type Dependency struct {
	Name string
	SHA1 string
	Size int64
}

// MavenManifest minimal representation of maven-metadata.xml
type MavenManifest struct {
	Versioning Versioning `xml:"versioning"`
}

// Versioning ...
type Versioning struct {
	Snapshot Snapshot `xml:"snapshot"`
}

// Snapshot ...
type Snapshot struct {
	Timestamp   string `xml:"timestamp"`
	BuildNumber int    `xml:"buildNumber"`
}

// NewBinaryDownloader returns a new BinaryDownloader of type Downloader
func NewBinaryDownloader(path string) Downloader {
	return &BinaryDownloader{
		Path: path,
	}
}

// DownloadArtifact prepare the binary artifact
func (n *BinaryDownloader) DownloadArtifact(_ *config.MavenGav) (Deliverable, error) {
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

// NewMavenDownloader MavenDownloader of type Downloader
func NewMavenDownloader(baseURL string, username string, password string) Downloader {
	return &MavenDownloader{
		baseURL:  baseURL,
		username: username,
		password: password,
	}
}

// DownloadArtifact downloads a maven artifact
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
			return deliverable, errors.Wrapf(err, "Unable to parse nexus url %s", n.baseURL)
		}
		//Set path
		u.Path = createMavenManifestPath(c)
		logrus.Infof("Downloading artifact from %s", u.String())

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
			return deliverable, errors.Wrapf(err, "Requested resource %s. XML decode failed", u.String())
		}
	}

	// Create download url
	u, err := url.Parse(n.baseURL)
	if err != nil {
		return deliverable, errors.Wrapf(err, "Unable to parse nexus url %s", n.baseURL)
	}
	// Set path
	u.Path = createDownloadPath(mavenManifest, c)
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

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return deliverable, errors.Errorf("Could not download artifact (Make sure you have deployed it!)"+
			". Status code %s , Location %s", resp.Status, resp.Request.URL)
	}

	fileName := createFileName(c, mavenManifest)

	dir, err := ioutil.TempDir("", "package")
	if err != nil {
		return deliverable, errors.Wrap(err, "Failed to create directory for artifact")
	}
	filePath := filepath.Join(dir, fileName)

	fileCreated, err := os.Create(filePath)
	if err != nil {
		return deliverable, errors.Wrapf(err, "Failed to create artifact file. Filepath=%s", filePath)
	}
	defer fileCreated.Close()

	_, err = io.Copy(fileCreated, resp.Body)
	if err != nil {
		return deliverable, errors.Wrapf(err, "Failed to write to artifact file. Filepath=%s", filePath)
	}
	deliverable.Path = filePath
	logrus.Debugf("Downloaded artifact to %s", deliverable.Path)

	hash, _ := hashFileSHA1(deliverable.Path)
	deliverable.SHA1 = hash

	return deliverable, nil

}

func createMavenManifestPath(c *config.MavenGav) string {
	groupID := strings.ReplaceAll(c.GroupID, ".", "/")
	return fmt.Sprintf("/repository/maven-intern/%s/%s/%s/maven-metadata.xml", groupID, c.ArtifactID, c.Version)
}

func createDownloadPath(manifest MavenManifest, c *config.MavenGav) string {
	groupID := strings.ReplaceAll(c.GroupID, ".", "/")
	artifact := createFileName(c, manifest)
	return fmt.Sprintf("/repository/maven-intern/%s/%s/%s/%s", groupID, c.ArtifactID, c.Version, artifact)
}

func createFileName(gav *config.MavenGav, manifest MavenManifest) string {
	versionWithoutSnapshot := strings.ReplaceAll(gav.Version, "SNAPSHOT", "")
	if gav.IsSnapshot() && (len(manifest.Versioning.Snapshot.Timestamp) > 0 && manifest.Versioning.Snapshot.BuildNumber > 0) {
		return fmt.Sprintf("%s-%s%s-%d%s", gav.ArtifactID, versionWithoutSnapshot, manifest.Versioning.Snapshot.Timestamp, manifest.Versioning.Snapshot.BuildNumber, getClassifierExt(gav))
	}
	return fmt.Sprintf("%s-%s%s", gav.ArtifactID, gav.Version, getClassifierExt(gav))
}

func getClassifierExt(c *config.MavenGav) string {
	if c.Classifier != "" {
		return fmt.Sprintf("-%s.%s", c.Classifier, c.Type)
	}
	return fmt.Sprintf(".%s", c.Type)
}

/*
Create app version. If not snapshot build, then return version from GAV.
Otherwise, create new snapshot version based on deliverable.
*/
func GetSnapshotTimestampVersion(gav config.MavenGav, deliverable Deliverable) string {
	if gav.IsSnapshot() {
		replacer := strings.NewReplacer(gav.ArtifactID+"-", "", "-"+string(gav.Classifier)+"."+string(gav.Type), "")
		version := "SNAPSHOT-" + replacer.Replace(path.Base(deliverable.Path))
		return version
	}
	return gav.Version
}
