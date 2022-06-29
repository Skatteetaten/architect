package prepare

import (
	"encoding/json"
	"github.com/skatteetaten/architect/v2/pkg/util"
	"io"
)

//Web :
type Web struct {
	ConfigurableProxy bool           `json:"configurableProxy"`
	Nodejs            Nodejs         `json:"nodejs"`
	WebApp            WebApp         `json:"webapp"`
	Gzip              nginxGzip      `json:"gzip"`
	Exclude           []string       `json:"exclude"`
	Locations         nginxLocations `json:"locations"`
}

//Nodejs :
type Nodejs struct {
	Main      string            `json:"main"`
	Overrides map[string]string `json:"overrides"`
}

//WebApp :
type WebApp struct {
	Content         string            `json:"content"`
	Path            string            `json:"path"`
	DisableTryfiles bool              `json:"disableTryfiles"`
	Headers         map[string]string `json:"headers"`
}

//OpenshiftConfig :
type OpenshiftConfig struct {
	Web Web `json:"web"`
}

func newRadishNginxConfig(docker *ImageMetadata, nginx *nginxfileData) util.WriterFunc {
	return func(writer io.Writer) error {
		data := OpenshiftConfig{
			Web: Web{
				ConfigurableProxy: nginx.ConfigurableProxy,
				Nodejs: Nodejs{
					Main:      docker.Main,
					Overrides: nginx.NginxOverrides,
				},
				WebApp: WebApp{
					Content:         docker.Static,
					Path:            nginx.Path,
					DisableTryfiles: !nginx.SPA,
					Headers:         nginx.ExtraStaticHeaders,
				},
				Gzip:      nginx.Gzip,
				Exclude:   nginx.Exclude,
				Locations: nginx.Locations,
			},
		}
		err := json.NewEncoder(writer).Encode(data)
		return err
	}
}
