package util

import (
	"github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	"os"
)

func ExtractBinaryFromStdIn() (string, error) {
	tmpfile, err := ioutil.TempFile("", "binarybuild-architect")
	defer tmpfile.Close()
	if err != nil {
		return "", errors.Wrap(err, "Error opening tmpfile")
	}
	stdin := os.Stdin
	if _, err := io.Copy(tmpfile, stdin); err != nil {
		return "", errors.Wrapf(err, "Error writing file %s", tmpfile)
	}
	logrus.Debugf("Using file %s", tmpfile.Name())
	return tmpfile.Name(), nil
}
