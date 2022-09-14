package trace

import (
	"fmt"
	"github.com/anchore/syft/syft"
	"github.com/anchore/syft/syft/pkg/cataloger"
	"github.com/anchore/syft/syft/sbom"
	source "github.com/anchore/syft/syft/source"
)

// use syft to discover packages + distro only
func ScanImage(buildFolder string) {

	imageUrl := "dir:" + buildFolder

	sbom, err := Scan(imageUrl)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(sbom)
}

func Scan(imageUrl string) (string, error) {

	input, err := source.ParseInput(imageUrl, "", false)
	if err != nil {
		return "", err
	}

	src, cleanup, err := source.New(*input, nil, nil)
	if err != nil {
		return "", err
	}
	defer cleanup()

	catalog, _, release, err := syft.CatalogPackages(src, cataloger.Config{
		Search: cataloger.SearchConfig{
			Scope: source.SquashedScope,
		},
	})
	if err != nil {
		return "", err
	}

	resultSbom := sbom.SBOM{
		Artifacts: sbom.Artifacts{
			PackageCatalog:    catalog,
			LinuxDistribution: release,
		},
		Source: src.Metadata,
	}

	bytes, err := syft.Encode(resultSbom, syft.FormatByID(syft.CycloneDxJSONFormatID))
	//bytes, err := syft.Encode(resultSbom, SporingsLoggerFormat.Format())

	return string(bytes), err
}
