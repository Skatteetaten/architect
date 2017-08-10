package npm

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
)

type Downloader interface {
	DownloadPackageJson(name string) (*PackageJson, error)
	DownloadTarball(url string) (string, error)
}

type LocalClient struct {
	Basedir string
}

func NewLocalRegistry(basedir string) Downloader {
	return &LocalClient{Basedir: basedir}
}

type BinaryDownloader struct {
	tarball string
	version string
}

func (m *BinaryDownloader) DownloadPackageJson(name string) (*PackageJson, error) {
	return &PackageJson{
		Versions: map[string]VersionedPackageJson{
			m.version: {},
		},
	}, nil
}

func (m *BinaryDownloader) DownloadTarball(name string) (string, error) {
	return m.tarball, nil
}

func NewBinaryBuildRegistry(pathToTarball string, version string) Downloader {
	return &BinaryDownloader{
		tarball: pathToTarball,
		version: version,
	}
}

func NewRemoteRegistry(url string) Downloader {
	return &RegistryClient{RegistryUrl: url}
}

func (n *LocalClient) DownloadPackageJson(name string) (*PackageJson, error) {
	b, err := ioutil.ReadFile(filepath.Join(n.Basedir, name+".json"))
	if err != nil {
		return nil, errors.Wrap(err, "Error reading file")
	}
	p := &PackageJson{}
	err = json.Unmarshal(b, p)
	return p, err
}

func (n *LocalClient) DownloadTarball(url string) (string, error) {
	filename := path.Base(url)
	return filepath.Join(n.Basedir, filename), nil
}

type RegistryClient struct {
	RegistryUrl string
}

func (n *RegistryClient) DownloadPackageJson(name string) (*PackageJson, error) {
	url := n.RegistryUrl + "/" + name
	logrus.Debugf("Getting package.json from %s", url)
	httpResponse, err := http.Get(url)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get package.json from %s", url)
	}
	defer httpResponse.Body.Close()

	if httpResponse.StatusCode != 200 {
		return nil, errors.Errorf("Error getting package.json. From %s status %s", url, httpResponse.Status)
	}
	packageJson := &PackageJson{}

	decoder := json.NewDecoder(httpResponse.Body)
	err = decoder.Decode(packageJson)
	if err != nil {
		return nil, errors.Wrap(err, "Error unmarshalling data")
	}
	return packageJson, nil
}

func (n *RegistryClient) DownloadTarball(url string) (string, error) {
	// Get the data
	resp, err := http.Get(url)
	if resp.StatusCode != 200 {
		return "", errors.Errorf("Could not get %s, %s", url, resp.Status)
	}
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if err != nil {
		return "", err
	}
	tmpfile, err := ioutil.TempFile("", "nodeapp")
	if err != nil {
		return "", errors.Wrap(err, "Error creating tmpfile for tarball")
	}
	defer tmpfile.Close()
	_, err = io.Copy(tmpfile, resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "Error saving tarball")
	}
	return tmpfile.Name(), nil
}

func ExtractTarball(pathToTarball string) (string, error) {
	tmpdir, err := ioutil.TempDir("", "nodejs-architect")
	tarball, err := os.Open(pathToTarball)
	if err != nil {
		return "", errors.Wrap(err, "Error opening tarball")
	}
	defer tarball.Close()
	gzipStream, err := gzip.NewReader(tarball)
	if err != nil {
		return "", err
	}

	defer gzipStream.Close()

	tarReader := tar.NewReader(gzipStream)

	for true {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		} else if err != nil {
			return "", errors.Wrapf(err, "Error extracting tarball")
		} else if header == nil {
			continue
		}

		name := header.Name

		target := filepath.Join(tmpdir, header.Name)

		dirname := filepath.Dir(target)
		switch header.Typeflag {
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil {
					return tmpdir, errors.Wrapf(err, "Error writing file %s", name)
				}
			}
		case tar.TypeReg: // = regular file
			ret, err := writeInternal(func() (string, error) {
				if _, err := os.Stat(dirname); err != nil {
					if err := os.MkdirAll(dirname, 0755); err != nil {
						return tmpdir, errors.Wrapf(err, "Error writing file %s", name)
					}
				}
				f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
				if err != nil {
					return tmpdir, errors.Wrapf(err, "Error writing file %s", name)
				}
				defer f.Close()

				// copy over contents

				if _, err := io.Copy(f, tarReader); err != nil {
					return tmpdir, errors.Wrapf(err, "Error writing file %s", name)
				}
				return tmpdir, nil
			})
			if err != nil {
				return ret, err
			}

		default:
			logrus.Infof("Dont support %s", header.Typeflag)
		}
	}
	return tmpdir, nil
}

//Probably a better way of doing this.. But need to have file.Close() called to prevent to many open fd's.
func writeInternal(deferrerd func() (string, error)) (string, error) {
	return deferrerd()
}

func FindPackageJsonInsideTarball(pathToTarball string) (*VersionedPackageJson, error) {
	tarball, err := os.Open(pathToTarball)
	if err != nil {
		return nil, errors.Wrap(err, "Error opening tarball")
	}
	defer tarball.Close()
	gzipStream, err := gzip.NewReader(tarball)
	if err != nil {
		return nil, err
	}

	defer gzipStream.Close()

	tarReader := tar.NewReader(gzipStream)

	for true {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		} else if err != nil {
			return nil, errors.Wrapf(err, "Error extracting tarball")
		} else if header == nil {
			continue
		}

		if header.Typeflag == tar.TypeReg && header.Name == "package/package.json" {
			v := &VersionedPackageJson{}
			err := json.NewDecoder(tarReader).Decode(v)
			if err != nil {
				return nil, errors.Wrap(err, "Error reading package.json")
			}
			return v, nil
		}
	}
	return nil, errors.New("Did not find any package.json in archive")
}
