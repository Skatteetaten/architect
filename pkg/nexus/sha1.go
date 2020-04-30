package nexus

import (
	"crypto/sha1"
	"encoding/hex"
	"io"
	"os"
)

func hashFileSHA1(filePath string) (string, error) {
	var SHA1String string

	file, err := os.Open(filePath)
	if err != nil {
		return SHA1String, err
	}

	defer file.Close()

	hash := sha1.New()

	if _, err := io.Copy(hash, file); err != nil {
		return SHA1String, err
	}

	hashInBytes := hash.Sum(nil)[:20]

	SHA1String = hex.EncodeToString(hashInBytes)

	return SHA1String, nil
}
