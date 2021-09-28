package config

import (
	"strings"
	"time"
)

type ApplicationType string

const (
	JavaLeveransepakke   ApplicationType = "JavaLeveransepakke"
	NodeJsLeveransepakke ApplicationType = "NodeJsLeveranse"
	DoozerLeveranse      ApplicationType = "DoozerLeveranse"
	NodeJs                               = "NODEJS"
	Doozer                               = "DOOZER"
)

type PackageType string

const (
	ZipPackaging PackageType = "zip"
	TgzPackaging PackageType = "tgz"
)

type Classifier string

const (
	Leveransepakke       Classifier = "Leveransepakke"
	Webleveransepakke    Classifier = "Webleveransepakke"
	Doozerleveransepakke Classifier = "Doozerleveransepakke"
)

type BuildType string

const (
	Snapshot BuildType = "Snapshot"
	Release  BuildType = "Release"
)

type Config struct {
	ApplicationType   ApplicationType
	ApplicationSpec   ApplicationSpec
	DockerSpec        DockerSpec
	BuilderSpec       BuilderSpec
	BinaryBuild       bool
	LocalBuild        bool
	TLSVerify         bool
	BuildTimeout      time.Duration
	NoPush            bool
	SporingsContext   string
	Sporingstjeneste  string
	OwnerReferenceUid string
	BuildType         BuildType
}

type NexusAccess struct {
	Username string
	Password string
	NexusURL string
}

func (n NexusAccess) IsValid() bool {
	return len(n.Username) > 0 && len(n.Password) > 0 && len(n.NexusURL) > 0
}
func (n NexusAccess) String() string {
	return "{Username:" + n.Username + " Password:****** NexusURL:" + n.NexusURL + "}"
}

type ApplicationSpec struct {
	MavenGav      MavenGav
	BaseImageSpec DockerBaseImageSpec
}

//GAV parametersclear
type MavenGav struct {
	ArtifactId string
	GroupId    string
	Version    string
	Classifier Classifier
	Type       PackageType
}

//Check if GAV is snapshot
func (m *MavenGav) IsSnapshot() bool {
	return strings.Contains(m.Version, "SNAPSHOT")
}

//Get name
func (m *MavenGav) Name() string {
	return strings.Join([]string{m.GroupId, m.ArtifactId, m.ArtifactId}, ":")
}

type DockerBaseImageSpec struct {
	BaseImage   string
	BaseVersion string
}

type DockerSpec struct {
	OutputRegistry       string
	OutputRepository     string
	InternalPullRegistry string
	PushExtraTags        PushExtraTags
	//This is the external docker registry where we check versions.
	ExternalDockerRegistry string
	//The tag to push to. This is only used for ImageStreamTags (as for now) and RETAG functionality
	TagWith   string
	RetagWith string
}

type BuilderSpec struct {
	Version string
}

type PushExtraTags struct {
	Latest bool
	Major  bool
	Minor  bool
	Patch  bool
}

// Generates the tags given the appversion and extra tag configuration. Don't do any filtering
func (m *PushExtraTags) ToStringValue() string {
	str := make([]string, 0, 5)
	if m.Major {
		str = append(str, "major")
	}
	if m.Minor {
		str = append(str, "minor")
	}
	if m.Patch {
		str = append(str, "patch")
	}
	if m.Latest {
		str = append(str, "latest")
	}
	return strings.Join(str, ",")
}

//Get external registry url without protocol
func (m DockerSpec) GetExternalRegistryWithoutProtocol() string {
	return strings.TrimPrefix(m.ExternalDockerRegistry, "https://")
}

//Get internal registry url without protocol
func (m DockerSpec) GetInternalPullRegistryWithoutProtocol() string {
	return strings.TrimPrefix(m.InternalPullRegistry, "https://")
}

//Parse extra tags
func ParseExtraTags(i string) PushExtraTags {
	p := PushExtraTags{}
	if strings.Contains(i, "major") {
		p.Major = true
	}
	if strings.Contains(i, "minor") {
		p.Minor = true
	}
	if strings.Contains(i, "patch") {
		p.Patch = true
	}
	if strings.Contains(i, "latest") {
		p.Latest = true
	}
	return p
}
