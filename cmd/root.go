package cmd

import (
	"fmt"
	"github.com/skatteetaten/architect/cmd/architect"
	"github.com/spf13/cobra"
	"os"
)

var cfgFile string

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "architect",
	Short: "Architect, the build tool for Applications",
	Long: `Architect is a tool for building Docker images in a standarized way.

For now, the following is supported:

- Java applications packaged as a zip, with a defined structure
- NodeJS application packaged as a zip with a defined structure
	`,
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	RootCmd.AddCommand(architect.Build)
	RootCmd.AddCommand(architect.Prepare)
	RootCmd.AddCommand(architect.TagAndPush)
	// Here you will define your flags and configuration settings.
	// Cobra supports Persistent Flags, which, if defined here,
	// will be global for your application.

	//RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.architect.yaml)")
	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	//RootCmd.Flags().BoolP("verbose", "v", false, "Verbose logging")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	//TODO
}
