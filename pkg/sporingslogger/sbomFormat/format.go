package sbomFormat

import (
	"encoding/json"
	"io"

	"github.com/anchore/syft/syft/sbom"
)

const ID sbom.FormatID = "github-0-json"

func Format() sbom.Format {
	return sbom.NewFormat(
		ID,
		func(writer io.Writer, sbom sbom.SBOM) error {
			bom := toPackageNodes(&sbom)
			bytes, err := json.Marshal(bom)
			if err != nil {
				return err
			}
			_, err = writer.Write(bytes)

			return err
		},
		nil,
		nil,
	)
}
