package prepare

import (
	"encoding/json"
	"github.com/skatteetaten/architect/v2/pkg/doozer/config"
	"github.com/skatteetaten/architect/v2/pkg/util"
	"io"
)

type typeVersion struct {
	Type    string `json:"Type"`
	Version string `json:"Version"`
}

type javaDescriptorData struct {
	Basedir               string   `json:"Basedir"`
	PathsToClassLibraries []string `json:"PathsToClassLibraries"`
	MainClass             string   `json:"MainClass"`
	ApplicationArgs       string   `json:"ApplicationArgs"`
	JavaOptions           string   `json:"JavaOptions"`
}

type javaDescriptor struct {
	typeVersion
	Data javaDescriptorData
}

type generalDescriptor struct {
	typeVersion
}

func newRadishDescriptor(meta *config.DeliverableMetadata, basedir string) util.WriterFunc {
	return func(writer io.Writer) error {
		if meta.Java != nil {
			desc := javaDescriptor{
				typeVersion: typeVersion{
					Type:    "Java",
					Version: "1",
				},
				Data: javaDescriptorData{
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
		desc := generalDescriptor{
			typeVersion: typeVersion{
				Type:    "General",
				Version: "1",
			},
		}
		err := json.NewEncoder(writer).Encode(desc)
		return err

	}
}
