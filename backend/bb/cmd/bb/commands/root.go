package commands

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/buildbeaver/buildbeaver/bb/cmd/bb/cli"
	"github.com/buildbeaver/buildbeaver/common/version"
)

const (
	DefaultConfigDir = "~/"
	ConfigFileName   = ".bb"
)

var (
	defaultConfigFilePath = fmt.Sprintf("%s%s.yml", DefaultConfigDir, ConfigFileName)
)

type GlobalConfig struct {
	Debug          bool
	JSON           bool
	ConfigFilePath string
}

var Global = &GlobalConfig{}

func init() {
	cobra.OnInitialize(initConfig)
	cobra.OnInitialize(initEnv)

	RootCmd.PersistentFlags().StringVarP(
		&Global.ConfigFilePath,
		"config",
		"c",
		defaultConfigFilePath,
		"The config file to use when executing commands.")

	RootCmd.PersistentFlags().BoolVarP(
		&Global.Debug,
		"debug",
		"d",
		false,
		"Enable verbose debug output.")

	RootCmd.PersistentFlags().BoolVarP(
		&Global.JSON,
		"json",
		"j",
		false,
		"Enable structured JSON output.")
}

func initEnv() {

}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cli.Exit(RootCmd.Execute())
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {

	if Global.ConfigFilePath != "" && Global.ConfigFilePath != defaultConfigFilePath {
		viper.SetConfigFile(Global.ConfigFilePath)
	} else {
		viper.SetConfigName(ConfigFileName)
		viper.AddConfigPath(DefaultConfigDir)
		viper.AddConfigPath(".")
	}

	viper.AutomaticEnv()

	// If a config file is found, read it in.
	err := viper.ReadInConfig()
	if err == nil {
		Global.ConfigFilePath = viper.ConfigFileUsed()
		cli.Stderr.Printf("Using config file: %s", viper.ConfigFileUsed())
	} else {
		switch err.(type) {
		case viper.ConfigFileNotFoundError:
		default:
			cli.Exit(fmt.Errorf("error loading config file (%s): %s", viper.ConfigFileUsed(), err))
		}
	}
}

var RootCmd = &cobra.Command{
	Use:     "bb",
	Short:   "BuildBeaver",
	Long:    `BuildBeaver`,
	Version: version.VersionToString(),
	PersistentPreRun: func(cmd *cobra.Command, args []string) {

	},
}
