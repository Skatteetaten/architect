package process

import (
	"crypto/sha256"
	"encoding/hex"
	"github.com/sirupsen/logrus"
	"github.com/skatteetaten/architect/v2/pkg/util"
	"io/ioutil"
	"os"
	"testing"
)

func TestDigestCalculations(t *testing.T) {

	outputPath := "testdata"

	t.Run("Checksum tar.gz layer", func(t *testing.T) {
		hasher := sha256.New()
		s, err := ioutil.ReadFile(outputPath + "/app-layer.tar.gz")
		hasher.Write(s)
		if err != nil {
			t.Fatal("Failed to read tar.gz file")
		}

		// sha256sum < app-layer.tar.gz
		digest := hex.EncodeToString(hasher.Sum(nil))
		if digest != "71063abc5e0fbe4408a23a409e856d0ee96e4d12e9ea4e69549db55944968c06" {
			t.Fatal("Not equal expected digest")
		}
		logrus.Info(digest)
	})

	t.Run("Test checksum layer content", func(t *testing.T) {
		file, err := os.Open(outputPath + "/app-layer.tar.gz")
		uncompressedStream := util.ExtractGz(file)

		hasher := sha256.New()
		tarContent, err := ioutil.ReadAll(uncompressedStream)
		if err != nil {
			t.Fatal("Failed to read TAR stream")
		}
		hasher.Write(tarContent)
		digest := hex.EncodeToString(hasher.Sum(nil))
		// gunzip < app-layer.tar.gz  | sha256sum
		if digest != "af884e2df0939611af49e8d59acffabb81c0045f3e6fddfe382971c9b509a5be" {
			t.Fatal("Not equal expected digest")
		}
	})
}
