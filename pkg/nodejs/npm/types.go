package npm

type VersionedPackageJson struct {
	Aurora AuroraApplication `json:"aurora"`
	Dist   struct {
		Shasum  string `json:"shasum"`
		Tarball string `json:"tarball"`
	} `json:"dist"`
	Maintainers []Maintainer `json:"maintainers"`
}

type Maintainer struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type PackageJson struct {
	Versions map[string]VersionedPackageJson `json:"versions"`
	Name     string                          `json:"name"`
	Time     *struct {
		Created  string `json:"created"`
		Modified string `json:"modified"`
	} `json:"time"`
}

type AuroraApplication struct {
	NodeJS NodeJSApplication `json:"nodejs"`
	Static string            `json:"static"`
}

type NodeJSApplication struct {
	Main    string `json:"main"`
	Waf     string `json:"waf"`
	Runtime string `json:"runtime"`
}
