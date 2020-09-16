package util

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
)

//CalculateDigestFromArchive calculates a sha256 hash
func CalculateDigestFromArchive(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", errors.Wrapf(err, "Failed to open file %s", path)
	}
	uncompressedStream := ExtractGz(file)

	hasher := sha256.New()
	tarContent, err := ioutil.ReadAll(uncompressedStream)
	if err != nil {
		return "", errors.Wrap(err, "Failed to read uncompressed stream")
	}
	hasher.Write(tarContent)
	digest := hex.EncodeToString(hasher.Sum(nil))
	return fmt.Sprintf("sha256:%s", digest), nil
}

//CalculateDigest of tar content
func CalculateDigest(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", errors.Wrapf(err, "Failed to open file %s", path)
	}
	hasher := sha256.New()
	tarContent, err := ioutil.ReadAll(file)
	if err != nil {
		return "", errors.Wrapf(err, "Failed to read file %s", path)
	}
	hasher.Write(tarContent)
	digest := hex.EncodeToString(hasher.Sum(nil))
	return fmt.Sprintf("sha256:%s", digest), nil
}

//CalculateSize of file
func CalculateSize(path string) (int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, errors.Wrapf(err, "Failed to open file %s", path)
	}

	info, err := file.Stat()
	if err != nil {
		return 0, errors.Wrapf(err, "Unable extract file stats")
	}

	return int(info.Size()), nil
}
