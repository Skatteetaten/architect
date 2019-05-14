package architect

import (
	"github.com/spf13/cobra"
)

var TagAndPush = &cobra.Command{

	Use:   "tag-and-push",
	Short: "Tag and push image to docker registry",
	Long:  `TODO`,
	Run: func(cmd *cobra.Command, args []string) {
		//TODO: Generate tags from GAV ?

	},
}
