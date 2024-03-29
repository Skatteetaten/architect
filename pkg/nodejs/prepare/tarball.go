package prepare

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/skatteetaten/architect/v2/pkg/util"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

// TODO: Kan vi bruke den fra utils ?
func extractTarball(pathToTarball string) (string, error) {
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
				if err := util.MkdirAllWithPermissions(target, 0755); err != nil {
					return tmpdir, errors.Wrapf(err, "Error writing file %s", name)
				}
			}
		case tar.TypeReg: // = regular file
			ret, err := writeInternal(func() (string, error) {
				if _, err := os.Stat(dirname); err != nil {
					if err := util.MkdirAllWithPermissions(dirname, 0755); err != nil {
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
			logrus.Infof("Dont support %c", header.Typeflag)
		}
	}
	return tmpdir, nil
}

// Probably a better way of doing this.. But need to have file.Close() called to prevent to many open fd's.
func writeInternal(deferrerd func() (string, error)) (string, error) {
	return deferrerd()
}

func findOpenshiftJSONInTarball(pathToTarball string) (*openshiftJSON, error) {
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

		if header.Typeflag == tar.TypeReg && header.Name == "package/metadata/openshift.json" {
			v := &openshiftJSON{}
			err := json.NewDecoder(tarReader).Decode(v)
			if err != nil {
				return nil, errors.Wrap(err, "Error reading openshift.json")
			}
			return v, nil
		}
	}
	return nil, errors.New("Did not find any openshift.json in archive. Wrong format?")
}
