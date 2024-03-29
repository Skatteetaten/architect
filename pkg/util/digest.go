package util

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	"os"
)

// CalculateDigestFromArchive calculates a sha256 hash
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

// CalculateDigestFromFile of tar content
func CalculateDigestFromFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", errors.Wrapf(err, "Failed to open file %s", path)
	}
	defer file.Close()

	hasher := sha256.New()
	_, err = io.Copy(hasher, file)
	if err != nil {
		return "", errors.Wrapf(err, "Failed to read file %s", path)
	}
	digest := hex.EncodeToString(hasher.Sum(nil))
	return fmt.Sprintf("sha256:%s", digest), nil
}

// CalculateDigest returns sha256:<base64 digest> string of the provided byte slice.
func CalculateDigest(data []byte) string {
	hasher := sha256.New()
	hasher.Write(data)
	digest := hex.EncodeToString(hasher.Sum(nil))
	return fmt.Sprintf("sha256:%s", digest)
}
