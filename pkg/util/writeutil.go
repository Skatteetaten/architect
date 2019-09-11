package util

import (
	"github.com/pkg/errors"
	"io"
	"os"
	"path"
	"path/filepath"
	"text/template"
)

type FileWriter func(WriterFunc, ...string) error

type WriterFunc func(io.Writer) error

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

func NewByteWriter(data []byte) WriterFunc {
	return func(writer io.Writer) error {
		n, err := writer.Write(data)
		if err != nil {
			return errors.Wrapf(err, "Could not write data. Wrote %d bytes", n)
		}
		return nil
	}
}

func NewFileWriter(targetFolder string) FileWriter {
	return func(writerFunc WriterFunc, fileAsPath ...string) error {
		fileAsPath = append(fileAsPath, "")
		copy(fileAsPath[1:], fileAsPath[0:])
		fileAsPath[0] = targetFolder
		fp := filepath.Join(fileAsPath...)
		os.MkdirAll(path.Dir(fp), os.ModeDir|0755)
		fileToWriteTo, err := os.Create(fp)
		if err != nil {
			return errors.Wrapf(err, "Error creating %+t", fileAsPath)
		}
		defer fileToWriteTo.Close()
		err = writerFunc(fileToWriteTo)
		if err != nil {
			return errors.Wrap(err, "Error error writing data")
		}
		err = fileToWriteTo.Sync()
		if err != nil {
			return errors.Wrapf(err, "Error writing %+t", fileAsPath)
		}
		return nil
	}
}
