package util

import (
	"archive/tar"
	"compress/gzip"
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func CompressTarGz(src string, folder string, destination string) (string, error) {
	name := folder + "-layer.tar.gz"
	file, err := os.Create(destination + "/" + name)
	if err != nil {
		logrus.Fatalln(err)
	}
	defer file.Close()

	gw := gzip.NewWriter(file)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	targetFolder := src + "/" + folder
	err = filepath.Walk(targetFolder,
		func(path string, info os.FileInfo, err error) error {
			// return on any error
			if err != nil {
				return err
			}
			isSymlink := info.Mode()&os.ModeSymlink == os.ModeSymlink

			link := path
			if isSymlink {
				link, err = handleSymlink(path)
				if err != nil {
					return err
				}
				link = "/" + strings.TrimPrefix(strings.Replace(link, src, "", -1), string(filepath.Separator))
			}

			// create a new dir/path header
			header, err := tar.FileInfoHeader(info, link)
			if err != nil {
				return err
			}

			header.Name = strings.TrimPrefix(strings.Replace(path, src, "", -1), string(filepath.Separator))

			// write the header
			if err := tw.WriteHeader(header); err != nil {
				return err
			}

			if !info.Mode().IsRegular() {
				return nil
			}

			// open files for taring
			f, err := os.Open(path)
			if err != nil {
				return err
			}

			// copy path data into tar writer
			if _, err := io.Copy(tw, f); err != nil {
				return err
			}

			f.Close()

			return nil

		})
	return name, err
}

//get the filepath for the symbolic link
func handleSymlink(path string) (string, error) {
	//read the link
	link, err := os.Readlink(path)
	if err != nil {
		return "", err
	}

	if !filepath.IsAbs(link) {
		logrus.Info("Links")
		link = filepath.Join(filepath.Dir(path), link)
	}

	return link, nil
}

func ExtractGz(gzipStream io.Reader) *gzip.Reader {
	uncompressedStream, err := gzip.NewReader(gzipStream)
	if err != nil {
		logrus.Fatal("ExtractTarGz: NewReader failed")
	}

	return uncompressedStream
}
