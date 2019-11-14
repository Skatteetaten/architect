package prepare

import "github.com/skatteetaten/architect/pkg/config/runtime"

type auroraApplication struct {
	NodeJS *nodeJSApplication `json:"nodejs"`
	//Deprecated
	Static            string               `json:"static"`
	Webapp            *webApplication      `json:"webapp"`
	ConfigurableProxy bool            	   `json:"configurableProxy"`
	Gzip            nginxGzip              `json:"gzip"`
	Locations       map[string]interface{} `json:"locations"`
	Exclude         []string               `json:"exclude"`
	//Deprecated
	Path string                            `json:"path"`
	//Deprecated
	SPA bool                               `json:"spa"`
}

type webApplication struct {
	StaticContent   string                 `json:"content"`
	Path            string                 `json:"path"`
	Headers         map[string]string      `json:"headers"`	
	DisableTryfiles bool                   `json:"disableTryfiles"`
}

type nodeJSApplication struct {
	Main      string            `json:"main"`
	Overrides map[string]string `json:"overrides"`
}

type openshiftJson struct {
	Aurora         auroraApplication `json:"web"`
	DockerMetadata dockerMetadata    `json:"docker"`
}

type dockerMetadata struct {
	Maintainer string            `json:"maintainer"`
	Labels     map[string]string `json:"labels"`
}

type PreparedImage struct {
	baseImage runtime.DockerImage
	Path      string
}

type probe struct {
	Include bool
	Port    int
}

type nginxLocations map[string]*nginxLocation

type nginxLocation struct {
	Headers map[string]string `json:"headers"`
	Gzip    nginxGzip         `json:"gzip"`
}

type nginxGzip struct {
	Use       string `json:"use"`
	MinLength int    `json:"min_length"`
	Vary      string `json:"vary"`
	Proxied   string `json:"proxied"`
	Types     string `json:"types"`
	Disable   string `json:"disable"`
}

type templateInput struct {
	Baseimage            string
	HasNodeJSApplication bool
	NginxOverrides       map[string]string
	ConfigurableProxy    bool
	Static               string
	SPA                  bool
	ExtraStaticHeaders   map[string]string
	Path                 string
	Labels               map[string]string
	Env                  map[string]string
	PackageDirectory     string
}
