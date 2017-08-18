package util

import (
	"github.com/pkg/errors"
	"io"
	"os"
	"path/filepath"
	"text/template"
)

type FileWriter func(WriterFunc, string) error

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

func NewFileWriter(targetFolder string) FileWriter {
	return func(writerFunc WriterFunc, filename string) error {
		fileToWriteTo, err := os.Create(filepath.Join(targetFolder, filename))
		if err != nil {
			errors.Wrapf(err, "Error creating %s", filename)
		}
		defer fileToWriteTo.Close()
		err = writerFunc(fileToWriteTo)
		if err != nil {
			return errors.Wrap(err, "Error processing template")
		}
		err = fileToWriteTo.Sync()
		if err != nil {
			return errors.Wrapf(err, "Error writing %s", filename)
		}
		return nil
	}
}
