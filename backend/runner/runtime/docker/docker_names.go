package docker

import (
	"fmt"
	"regexp"
)

var (
	DefaultDockerPrefix = "buildbeaver"
	BBDockerPrefix      = "bb" // do not include a dash in this name; RegEx assumes dash is end of prefix
	dockerJobRunSuffix  = "run"
	dockerServiceSuffix = "service"
	dockerNetworkSuffix = "network"
)

var (
	dockerPrefix              string
	jobContainerNameRegex     *regexp.Regexp
	serviceContainerNameRegex *regexp.Regexp
	networkNameRegex          *regexp.Regexp
)

func init() {
	SetDockerPrefix(DefaultDockerPrefix)
}

// SetDockerPrefix changes the prefix used for docker names (e.g. for containers, networks) to be different
// from DefaultDockerPrefix. This can enable different sets of docker names to be used for different commands
// so that they don't try to clean up each other's resources (e.g. for bb vs runners).
func SetDockerPrefix(prefix string) {
	dockerPrefix = prefix

	jobContainerNameRegexStr := "^" + dockerPrefix + "-[a-zA-Z0-9\\._-]+-" + dockerJobRunSuffix + "$"
	serviceContainerNameRegexStr := "^" + dockerPrefix + "-[a-zA-Z0-9\\._-]+-" + dockerServiceSuffix + "-[a-zA-Z0-9\\._-]+$"
	networkNameRegexStr := "^" + dockerPrefix + "-[a-zA-Z0-9\\._-]+-" + dockerNetworkSuffix + "$"

	jobContainerNameRegex = regexp.MustCompile(jobContainerNameRegexStr)
	serviceContainerNameRegex = regexp.MustCompile(serviceContainerNameRegexStr)
	networkNameRegex = regexp.MustCompile(networkNameRegexStr)
}

// makeContainerNameForJob makes a name for the docker container used for running a job's (step) commands,
// using the information in the specified docker config.
func makeContainerNameForJob(config *Config) string {
	return fmt.Sprintf("%s-%s-%s", dockerPrefix, config.RuntimeID, dockerJobRunSuffix)
}

// isContainerNameForJob returns true if the specified string is a valid container name for a docker container
// created by BuildBeaver to run a job.
func isContainerNameForJob(name string) bool {
	return jobContainerNameRegex.MatchString(name)
}

// makeContainerNameForService makes a name for the docker container running a service.
func makeContainerNameForService(serviceManagerConfig *ServiceManagerConfig, serviceConfig *ServiceConfig) string {
	return fmt.Sprintf("%s-%s-%s-%s", dockerPrefix, serviceManagerConfig.RuntimeID, dockerServiceSuffix, serviceConfig.Name)
}

// isContainerNameForJob returns true if the specified string is a valid container name for a docker container
// created by BuildBeaver to run a service.
func isContainerNameForService(name string) bool {
	return serviceContainerNameRegex.MatchString(name)
}

// makeNetworkName makes a name for the docker network connecting a job and its services, using the information
// in the specified docker config.
func makeNetworkName(config *Config) string {
	return fmt.Sprintf("%s-%s-%s", dockerPrefix, config.RuntimeID, dockerNetworkSuffix)
}

// isNetworkName returns true if the specified string is a valid network name for a docker network created by BuildBeaver.
func isNetworkName(name string) bool {
	return networkNameRegex.MatchString(name)
}
