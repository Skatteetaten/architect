package util

import (
	"github.com/pkg/errors"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"text/template"
)

//FileWriter function
type FileWriter func(WriterFunc, ...string) error

//WriterFunc function
type WriterFunc func(io.Writer) error

//NewTemplateWriter wrapper
func NewTemplateWriter(input interface{}, templatename string, templateString string) WriterFunc {
	return func(writer io.Writer) error {
		tmpl, err := template.New(templatename).Parse(templateString)
		if err != nil {
			return errors.Wrap(err, "Error parsing template")
		}
		err = tmpl.Execute(writer, input)
		if err != nil {
			return errors.Wrap(err, "Error processing template")
		}
		return nil
	}
}

// MkdirAllWithPermissions works like os.MkdirAll, but will set permissions even if the folder already exists.
func MkdirAllWithPermissions(path string, perm fs.FileMode) error {
	if err := os.MkdirAll(path, perm); err != nil {
		return errors.Wrapf(err, "Failed to create folder %s", path)
	}
	// In case the directory already existed, we adjust the permissions:
	if err := os.Chmod(path, perm); err != nil {
		return errors.Wrapf(err, "Failed to set permissions for folder %s", path)
	}
	return nil
}

// NewByteWriter wrapper
func NewByteWriter(data []byte) WriterFunc {
	return func(writer io.Writer) error {
		n, err := writer.Write(data)
		if err != nil {
			return errors.Wrapf(err, "Could not write data. Wrote %d bytes", n)
		}
		return nil
	}
}

//NewFileWriter wrapper
func NewFileWriter(targetFolder string) FileWriter {
	return func(writerFunc WriterFunc, fileAsPath ...string) error {
		fileAsPath = append(fileAsPath, "")
		copy(fileAsPath[1:], fileAsPath[0:])
		fileAsPath[0] = targetFolder
		fp := filepath.Join(fileAsPath...)
		MkdirAllWithPermissions(path.Dir(fp), os.ModeDir|0755)
		fileToWriteTo, err := os.Create(fp)
		if err != nil {
			return errors.Wrapf(err, "Error creating %v", fileAsPath)
		}
		defer fileToWriteTo.Close()
		err = writerFunc(fileToWriteTo)
		if err != nil {
			return errors.Wrap(err, "Error error writing data")
		}
		err = fileToWriteTo.Sync()
		if err != nil {
			return errors.Wrapf(err, "Error writing %v", fileAsPath)
		}
		return nil
	}
}
