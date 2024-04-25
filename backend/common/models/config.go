package models

import "strings"

const ConfigFileName = "buildbeaver.jsonnet"

const (
	ConfigTypeYAML    ConfigType = "yaml"
	ConfigTypeJSON    ConfigType = "json"
	ConfigTypeJSONNET ConfigType = "jsonnet"
	// ConfigTypeNoConfig (an empty string) is used to indicate that there is no configuration file present.
	ConfigTypeNoConfig ConfigType = ""
	// ConfigTypeInvalid is used to record that an invalid config was found, e.g. if the config is too long.
	// A more detailed error message should be stored in the config data.
	ConfigTypeInvalid ConfigType = "invalid"
	// ConfigTypeUnknown indicates that a config file of an unknown type was found.
	ConfigTypeUnknown ConfigType = "unknown"
)

type ConfigType string

func (s ConfigType) Valid() bool {
	return string(s) != ""
}

func (s ConfigType) String() string {
	return string(s)
}

func (s *ConfigType) Scan(src interface{}) error {
	if src == nil {
		*s = ConfigTypeNoConfig
		return nil
	}
	t := src.(string)
	switch strings.ToLower(t) {
	case string(ConfigTypeYAML):
		*s = ConfigTypeYAML
	case string(ConfigTypeJSON):
		*s = ConfigTypeJSON
	case string(ConfigTypeJSONNET):
		*s = ConfigTypeJSONNET
	case string(ConfigTypeNoConfig):
		*s = ConfigTypeNoConfig
	case string(ConfigTypeInvalid):
		*s = ConfigTypeInvalid
	default:
		*s = ConfigTypeUnknown
	}
	return nil
}
