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

	var resourceUrl string
	if useNexus3 {
		resourceUrl, err = n.createNexus3URL(c)
		if err != nil {
			return deliverable, errors.Wrapf(err, "Failed to create Nexus 3 url for GAV %+v", c)
		}
	} else {
		resourceUrl, err = n.createURL(c)
		if err != nil {
			return deliverable, errors.Wrapf(err, "Failed to create Nexus url for GAV %+v", c)
		}
	}
	logrus.Debugf("Downloading artifact from %s", resourceUrl)

	req, err := http.NewRequest("GET", resourceUrl, nil)
	if err != nil {
		return deliverable, errors.Wrapf(err, "Failed to create request for Nexus url %s", resourceUrl)
	}

	if useNexus3 && (na == nil || na.Username == "" || na.Password == "") {
		return deliverable, errors.Wrap(err, "Missing Nexus credentials for Nexus 3")
	}
	if na != nil {
		req.SetBasicAuth(na.Username, na.Password)
	}

	httpResponse, err := http.DefaultClient.Do(req)
	if err != nil {
		return deliverable, errors.Wrapf(err, "Failed to get artifact from Nexus %s", resourceUrl)
	}
	defer httpResponse.Body.Close()

	if httpResponse.StatusCode != http.StatusOK {
		return deliverable, errors.Errorf("Could not download artifact (Make sure you have deployed it!)"+
			". Status code %s ", httpResponse.Status)
	}

	var fileName string
	if useNexus3 {
		// No content-disposition from nexus3 - using alternate filename composition
		if string(c.Classifier) == "" {
			return deliverable, errors.Errorf("Missing maven Classifier")
		}
		if string(c.Type) == "" {
			return deliverable, errors.Errorf("Missing maven Type")
		}
		fileName = string(c.Classifier) + "." + string(c.Type)

	} else {
		contentDisposition := httpResponse.Header.Get("content-disposition")

		if len(contentDisposition) <= 0 {
			return deliverable, errors.Errorf("No content-disposition in response header")
		}

		_, params, err := mime.ParseMediaType(contentDisposition)
		if err != nil {
			return deliverable, errors.Wrap(err, "Failed to parse content-disposition")
		}
		fileName = params["filename"]
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
