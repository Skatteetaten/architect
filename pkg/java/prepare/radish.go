package prepare

import (
	"encoding/json"
	"github.com/skatteetaten/architect/pkg/java/config"
	"github.com/skatteetaten/architect/pkg/util"
	"io"
)

type Type struct {
	Type    string `json:"Type"`
	Version string `json:"Version"`
}

type JavaDescriptorData struct {
	Basedir               string   `json:"Basedir"`
	PathsToClassLibraries []string `json:"PathsToClassLibraries"`
	MainClass             string   `json:"MainClass"`
	ApplicationArgs       string   `json:"ApplicationArgs"`
	JavaOptions           string   `json:"JavaOptions"`
}

type JavaDescriptor struct {
	Type
	Data JavaDescriptorData
}

func newRadishDescriptor(meta *config.DeliverableMetadata, basedir string) util.WriterFunc {
	return func(writer io.Writer) error {
		desc := JavaDescriptor{
			Type: Type{
				Type:    "Java",
				Version: "1",
			},
			Data: JavaDescriptorData{
				Basedir:               basedir,
				PathsToClassLibraries: []string{"lib", "repo"},
				MainClass:             meta.Java.MainClass,
				ApplicationArgs:       meta.Java.ApplicationArgs,
				JavaOptions:           meta.Java.JvmOpts,
			},
		}

		err := json.NewEncoder(writer).Encode(desc)
		return err
	}
}
