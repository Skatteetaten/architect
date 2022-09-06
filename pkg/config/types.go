package config

import (
	"strings"
	"time"
)

// ApplicationType type string
type ApplicationType string

const (
	// JavaLeveransepakke ApplicationType
	JavaLeveransepakke ApplicationType = "JavaLeveransepakke"
	// NodeJsLeveransepakke ApplicationType
	NodeJsLeveransepakke ApplicationType = "NodeJsLeveranse"
	// DoozerLeveranse      ApplicationType
	DoozerLeveranse ApplicationType = "DoozerLeveranse"
	// NodeJs string
	NodeJs string = "NODEJS"
	// Doozer string
	Doozer string = "DOOZER"
)

// PackageType type string
type PackageType string

const (
	// ZipPackaging PackageType
	ZipPackaging PackageType = "zip"
	// TgzPackaging PackageType
	TgzPackaging PackageType = "tgz"
)

// Classifier string
type Classifier string

const (
	// Leveransepakke Classifier
	Leveransepakke Classifier = "Leveransepakke"
	// Webleveransepakke Classifier
	Webleveransepakke Classifier = "Webleveransepakke"
	// Doozerleveransepakke Classifier
	Doozerleveransepakke Classifier = "Doozerleveransepakke"
)

// BinaryBuildType type string
type BinaryBuildType string

const (
	// Snapshot BinaryBuildType
	Snapshot BinaryBuildType = "Snapshot"
)

// Config contains the build config
type Config struct {
	ApplicationType    ApplicationType
	ApplicationSpec    ApplicationSpec
	DockerSpec         DockerSpec
	BuilderSpec        BuilderSpec
	BinaryBuild        bool
	LocalBuild         bool
	TLSVerify          bool
	BuildTimeout       time.Duration
	NoPush             bool
	Sporingstjeneste   string
	OwnerReferenceUUID string
	BinaryBuildType    BinaryBuildType
	NexusIQReportURL   string
}

// NexusAccess nexus url and nexus credentials
type NexusAccess struct {
	Username string
	Password string
	NexusURL string
}

// IsValid check username, password and url is set
func (n NexusAccess) IsValid() bool {
	return len(n.Username) > 0 && len(n.Password) > 0 && len(n.NexusURL) > 0
}

// String return as string
func (n NexusAccess) String() string {
	return "{Username:" + n.Username + " Password:****** NexusURL:" + n.NexusURL + "}"
}

// ApplicationSpec config
type ApplicationSpec struct {
	MavenGav      MavenGav
	BaseImageSpec DockerBaseImageSpec
}

// MavenGav GAV parameters
type MavenGav struct {
	ArtifactID string
	GroupID    string
	Version    string
	Classifier Classifier
	Type       PackageType
}

// IsSnapshot Check if GAV is snapshot
func (m *MavenGav) IsSnapshot() bool {
	return strings.HasSuffix(m.Version, "SNAPSHOT")
}

// Name Get name
func (m *MavenGav) Name() string {
	return strings.Join([]string{m.GroupID, m.ArtifactID, m.ArtifactID}, ":")
}

// DockerBaseImageSpec config
type DockerBaseImageSpec struct {
	BaseImage   string
	BaseVersion string
}

// DockerSpec config
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

// BuilderSpec config
type BuilderSpec struct {
	Version string
}

// PushExtraTags config
type PushExtraTags struct {
	Latest bool
	Major  bool
	Minor  bool
	Patch  bool
}

// ToStringValue Generates the tags given the appversion and extra tag configuration. Don't do any filtering
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

// GetExternalRegistryWithoutProtocol Get external registry url without protocol
func (m DockerSpec) GetExternalRegistryWithoutProtocol() string {
	return strings.TrimPrefix(m.ExternalDockerRegistry, "https://")
}

// GetInternalPullRegistryWithoutProtocol Get internal registry url without protocol
func (m DockerSpec) GetInternalPullRegistryWithoutProtocol() string {
	return strings.TrimPrefix(m.InternalPullRegistry, "https://")
}

// ParseExtraTags parse extra tags
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
