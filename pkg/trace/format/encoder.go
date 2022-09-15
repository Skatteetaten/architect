package format

import (
	"fmt"
	"github.com/mholt/archiver/v3"
	"github.com/sirupsen/logrus"
	"strings"

	"github.com/anchore/syft/syft/pkg"
	"github.com/anchore/syft/syft/sbom"
	"github.com/anchore/syft/syft/source"
)

// isArchive returns true if the path appears to be an archive
func isArchive(path string) bool {
	_, err := archiver.ByExtension(path)
	return err == nil
}

// toPath Generates a string representation of the package location, optionally including the layer hash
func toPath(s source.Metadata, p pkg.Package) string {
	inputPath := strings.TrimPrefix(s.Path, "./")
	if inputPath == "." {
		inputPath = ""
	}
	locations := p.Locations.ToSlice()
	if len(locations) > 0 {
		location := locations[0]
		packagePath := location.RealPath
		if location.VirtualPath != "" {
			packagePath = location.VirtualPath
		}
		packagePath = strings.TrimPrefix(packagePath, "/")
		switch s.Scheme {
		case source.ImageScheme:
			return packagePath
		case source.FileScheme:
			if isArchive(inputPath) {
				return fmt.Sprintf("%s:/%s", inputPath, packagePath)
			}
			return inputPath
		case source.DirectoryScheme:
			if inputPath != "" {
				return fmt.Sprintf("%s/%s", inputPath, packagePath)
			}
			return packagePath
		}
	}
	return fmt.Sprintf("%s%s", inputPath, s.ImageMetadata.UserInput)
}

// toGithubManifests manifests, each of which represents a specific location that has dependencies
func toGithubManifests(s *sbom.SBOM) []PackageNode {
	var packages []PackageNode
	for _, p := range s.Artifacts.PackageCatalog.Sorted() {
		path := toPath(s.Source, p)
		checksumAlgorithm, checksumValue := getChecksums(p)

		packageNode := PackageNode{
			PackageURL:        p.PURL,
			Name:              p.Name,
			DependencyId:      p.ID(),
			Version:           p.Version,
			SourceLocation:    path,
			ChecksumAlgorithm: checksumAlgorithm,
			ChecksumValue:     checksumValue,
		}
		packages = append(packages, packageNode)
	}
	return packages
}

// The code for getting Checksum is taken from the spdx module
func getChecksums(p pkg.Package) (string, string) {
	// we generate digest for some Java packages
	// see page 33 of the spdx specification for 2.2
	// spdx.github.io/spdx-spec/package-information/#710-package-checksum-field

	var checksumValue string
	var checksumAlgorithm string

	if p.MetadataType == pkg.JavaMetadataType {
		javaMetadata := p.Metadata.(pkg.JavaMetadata)
		if len(javaMetadata.ArchiveDigests) > 0 {
			checksumValue = javaMetadata.ArchiveDigests[0].Value
			checksumAlgorithm = javaMetadata.ArchiveDigests[0].Algorithm
		}
		if len(javaMetadata.ArchiveDigests) > 1 {
			logrus.Info("Only one checksum is handeled by Sporingslogger")
		}
	}

	return checksumAlgorithm, checksumValue
}
