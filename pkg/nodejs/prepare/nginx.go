package prepare

type NginxfileData struct {
	HasNodeJSApplication bool
	ConfigurableProxy    bool
	NginxOverrides       map[string]string
	Path                 string
	ExtraStaticHeaders   map[string]string
	SPA                  bool
	Content              string
	Exclude              []string
	Gzip                 nginxGzip
	Locations            nginxLocations
}
