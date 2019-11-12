package prepare

import (
	"encoding/json"
	"github.com/skatteetaten/architect/pkg/util"
	"io"
)

//Web :
type Web struct {
	ConfigurableProxy bool     `json:"configurableProxy"`
	Nodejs            Nodejs   `json:"nodejs"`
	WebApp            WebApp   `json:"webapp"`
	Exclude           []string `json:"exclude"`
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

func UnmarshallOpenshiftConfig(buffer io.Reader) (OpenshiftConfig, error) {
	var data OpenshiftConfig
	err := json.NewDecoder(buffer).Decode(&data)
	return data, err
}

func newRadishNginxConfig(docker *DockerfileData, nginx *NginxfileData) util.WriterFunc {
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
				Exclude: nginx.Exclude,
			},
		}
		err := json.NewEncoder(writer).Encode(data)
		return err
	}
}
