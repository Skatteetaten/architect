package util

import (
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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
		return "", errors.Wrapf(err, "Error writing file %s", tmpfile.Name())
	}
	logrus.Debugf("Using file %s", tmpfile.Name())
	return tmpfile.Name(), nil
}

func ExtractBinaryFromFile(file string) (string, error) {
	tmpfile, err := ioutil.TempFile("", "binarybuild-architect")
	defer tmpfile.Close()
	if err != nil {
		return "", errors.Wrap(err, "Error opening tmpfile")
	}

	source, err := os.Open(file)
	if err != nil {
		return "", errors.Wrapf(err, "Unable to open file %s", file)
	}
	defer source.Close()

	if _, err := io.Copy(tmpfile, source); err != nil {
		return "", errors.Wrapf(err, "Error writing file %s", tmpfile.Name())
	}

	logrus.Debugf("Using file %s", tmpfile.Name())
	return tmpfile.Name(), nil

}
