package commands

import (
	"github.com/buildbeaver/buildbeaver/common/version"
	"github.com/spf13/cobra"

	"github.com/buildbeaver/buildbeaver/server/cmd/bb-tools/cli"
)

type GlobalConfig struct {
	Debug bool
}

var Global = &GlobalConfig{}

func init() {
	cobra.OnInitialize(initConfig)

	RootCmd.PersistentFlags().BoolVarP(
		&Global.Debug,
		"debug",
		"d",
		false,
		"Enable debug-level log output.")

}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cli.Exit(RootCmd.Execute())
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
}

var RootCmd = &cobra.Command{
	Use:     "bb-tools command",
	Short:   "BuildBeaver tools",
	Long:    `BuildBeaver tools`,
	Version: version.VersionToString(),
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
	},
}
