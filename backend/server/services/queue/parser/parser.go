package parser

import (
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/google/go-jsonnet"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"

	"github.com/buildbeaver/buildbeaver/common/models"
)

var (
	// YAMLBuildConfigFileNames contains a list of all build config file names
	// that represent a YAML formatted config file in the root of a git repo.
	YAMLBuildConfigFileNames = []string{
		".buildbeaver.yaml",
		"buildbeaver.yaml",
		".buildbeaver.yml",
		"buildbeaver.yml",
	}

	// JSONBuildConfigFileNames contains a list of all build config file names
	// that represent a JSON formatted config file in the root of a git repo.
	JSONBuildConfigFileNames = []string{
		".buildbeaver.json",
		"buildbeaver.json",
	}

	// JSONNETBuildConfigFileNames contains a list of all build config file names
	// that represent a JSONNET formatted config file in the root of a git repo.
	JSONNETBuildConfigFileNames = []string{
		".buildbeaver.jsonnet",
		"buildbeaver.jsonnet",
	}
)

var (
	// jobNameRegex defines the format of the job name field, with an optional workflow followed by a job name
	jobNameRegex = regexp.MustCompile(`(?im)^(?:([a-zA-Z0-9_-]+)\.)?([a-zA-Z0-9_*-]+)$`)

	// Job dependencies include an optional workflow, followed by a mandatory job name, then optionally 'artifacts'
	// (for all artifacts) or an artifact name (for a single artifact)
	// Note that sometimes the shorthand syntax can be ambiguous with the full syntax. In this case the full syntax
	// always wins, and the full syntax is never ambiguous. Examples:
	// 'jobs.myjobname' could mean a workflow called 'jobs' or the full jobs syntax for a job named 'myjobname'
	// 'name.artifacts' could mean a workflow 'name' and job called 'artifacts' or just job 'name' artifacts
	// 'jobs.artifacts' could be a job called 'jobs', or a job called 'artifacts' (full syntax)
	jobDependsOnOneArtifactFromJobRegex           = regexp.MustCompile(`(?im)^(?:workflow\.([a-zA-Z0-9_-]+)\.)?jobs\.([a-zA-Z0-9_*-]+)\.artifacts\.([a-zA-Z0-9_-]+)$`)
	jobDependsOnAllArtifactsFromJobRegex          = regexp.MustCompile(`(?im)^(?:workflow\.([a-zA-Z0-9_-]+)\.)?jobs\.([a-zA-Z0-9_*-]+)\.artifacts$`)
	jobDependsOnJobRegex                          = regexp.MustCompile(`(?im)^(?:workflow\.([a-zA-Z0-9_-]+)\.)?jobs\.([a-zA-Z0-9_*-]+)$`)
	jobDependsOnAllArtifactsFromJobShorthandRegex = regexp.MustCompile(`(?im)^(?:([a-zA-Z0-9_-]+)\.)?([a-zA-Z0-9_*-]+)\.artifacts$`)
	jobDependsOnJobShorthandRegex                 = regexp.MustCompile(`(?im)^(?:([a-zA-Z0-9_-]+)\.)?([a-zA-Z0-9_*-]+)$`)
)

// buildDefinitionVersionedParser is an object capable of parsing a specific version of a build definition.
type buildDefinitionVersionedParser interface {
	Parse(topLevelElement map[string]interface{}) (*models.BuildDefinition, error)
}

// ParserLimits provides a parser with information on limits to check while parsing. If the data goes beyond
// any limit then parsing should fail.
type ParserLimits struct {
	// MaxJobsPerBuild is the maximum number of steps allowed in any single job. Any build definition containing
	// a job with more than this number of steps will be rejected.
	MaxStepsPerJob int
}

type BuildDefinitionParser struct {
	limits ParserLimits
}

func NewBuildDefinitionParser(limits ParserLimits) *BuildDefinitionParser {
	return &BuildDefinitionParser{
		limits: limits,
	}
}

// Parse parses a raw build config.
func (s *BuildDefinitionParser) Parse(config []byte, configType models.ConfigType) (*models.BuildDefinition, error) {
	var (
		err   error
		raw   interface{}
		build *models.BuildDefinition
	)
	switch configType {
	case models.ConfigTypeYAML:
		raw, err = s.parseFromYAML(config)
	case models.ConfigTypeJSON:
		raw, err = s.parseFromJSON(config)
	case models.ConfigTypeJSONNET:
		raw, err = s.parseFromJSONNET(config)
	case models.ConfigTypeNoConfig:
		return nil, errors.Errorf("error: no build configuration file was found")
	case models.ConfigTypeInvalid:
		return nil, s.getErrorForInvalidConfig(config)
	default:
		return nil, errors.Errorf("error: unsupported build configuration type: %s", configType)
	}
	if err != nil {
		return nil, errors.Wrapf(err, "error unmarshalling build definition from %s", configType)
	}

	// All versions must have a top-level object rather than an array.
	topLevelElement, ok := raw.(map[string]interface{})
	if !ok {
		return nil, errors.Errorf("error parsing build definition: must contain a top-level object: %T", topLevelElement)
	}

	const defaultVersion = "DEFAULT_VERSION"
	version := defaultVersion
	rVersion, ok := topLevelElement["version"]
	if ok {
		// normalizeMapValues() turns all scalar data types into strings, including float/integer version numbers
		version, ok = rVersion.(string)
		if !ok {
			return nil, errors.Errorf("error parsing build definition: expected 'version' field to be a string but found: %T", rVersion)
		}
	}

	// Create a parser specific to the version to parse the rest of the data
	var parser buildDefinitionVersionedParser
	switch version {
	case "0.2", defaultVersion:
		parser = newBuildDefinitionParserV02(s.limits)
	case "0.3":
		parser = newBuildDefinitionParserV03(s.limits)
	case "1.0", "1":
		// TODO: Before release, define version 1.0 as the latest and make this the default by moving defaultVersion to this line
		return nil, errors.Errorf("error parsing build definition: version %s has not been defined in pre-release version", version)
	default:
		return nil, errors.Errorf("error parsing build definition: version %s not supported", version)
	}

	build, err = parser.Parse(topLevelElement)
	if err != nil {
		return nil, fmt.Errorf("error parsing build definition: %w", err)
	}

	return build, nil
}

func (s *BuildDefinitionParser) parseFromYAML(config []byte) (interface{}, error) {
	var raw interface{}
	err := yaml.Unmarshal(config, &raw)
	if err != nil {
		return nil, errors.Wrap(err, "error unmarshalling yml")
	}
	raw = s.normalizeMapValues(raw)
	return raw, nil
}

func (s *BuildDefinitionParser) parseFromJSON(config []byte) (interface{}, error) {
	var raw interface{}
	err := json.Unmarshal(config, &raw)
	if err != nil {
		return nil, errors.Wrap(err, "error unmarshalling json")
	}
	return raw, nil
}

func (s *BuildDefinitionParser) parseFromJSONNET(config []byte) (interface{}, error) {
	vm := jsonnet.MakeVM()
	json, err := vm.EvaluateSnippet(models.ConfigFileName, string(config[:]))
	if err != nil {
		return nil, errors.Wrap(err, "error parsing jsonnet")
	}
	return s.parseFromJSON([]byte(json))
}

// parseFromInvalid returns a suitable error message, given an invalid build configuration
func (s *BuildDefinitionParser) getErrorForInvalidConfig(config []byte) error {
	if len(config) == 0 {
		return errors.Errorf("error: invalid build configuration")
	}

	// For an invalid config, the config itself has been replaced with an error message
	message := string(config)
	if len(message) > 100 {
		message = message[:100]
	}

	return errors.Errorf("error: %s", message)
}

// normalizeMapValues iterates through all properties (including nested properties)
// of an object and converts all map[interface{}]interface{} that have a string key
// to map[string]interface{}. This is intended to be used to normalize the output of
// the yaml parser, to make it consistent with the JSON parser in the go standard lib.
func (s *BuildDefinitionParser) normalizeMapValues(v interface{}) interface{} {
	switch v := v.(type) {
	case []interface{}:
		return s.normalizeInterfaceArray(v)
	case map[interface{}]interface{}:
		return s.cleanupInterfaceMap(v)
	case string:
		return v
	default:
		// This will convert integers, floats and booleans to strings
		return fmt.Sprintf("%v", v)
	}
}

func (s *BuildDefinitionParser) normalizeInterfaceArray(in []interface{}) []interface{} {
	res := make([]interface{}, len(in))
	for i, v := range in {
		res[i] = s.normalizeMapValues(v)
	}
	return res
}

func (s *BuildDefinitionParser) cleanupInterfaceMap(in map[interface{}]interface{}) map[string]interface{} {
	res := make(map[string]interface{})
	for k, v := range in {
		res[fmt.Sprintf("%v", k)] = s.normalizeMapValues(v)
	}
	return res
}

// WorkflowDependencyFromString returns the names of the workflow and job in the given dependency string.
// An empty workflow name will be returned if no workflow was mentioned in the dependency.
func WorkflowDependencyFromString(dependency string) models.NodeFQN {
	var (
		workflow = "" // default is empty string
		jobName  = "" // default is empty string
		match    []string
	)
	if match = jobDependsOnOneArtifactFromJobRegex.FindStringSubmatch(dependency); match != nil {
		workflow = match[1]
		jobName = match[2]
	} else if match = jobDependsOnAllArtifactsFromJobRegex.FindStringSubmatch(dependency); match != nil {
		workflow = match[1]
		jobName = match[2]
	} else if match = jobDependsOnAllArtifactsFromJobShorthandRegex.FindStringSubmatch(dependency); match != nil {
		workflow = match[1]
		jobName = match[2]
	} else if match = jobDependsOnJobRegex.FindStringSubmatch(dependency); match != nil {
		workflow = match[1]
		jobName = match[2]
	} else if match = jobDependsOnJobShorthandRegex.FindStringSubmatch(dependency); match != nil {
		workflow = match[1]
		jobName = match[2]
	}

	return models.NewNodeFQNForJob(models.ResourceName(workflow), models.ResourceName(jobName))
}
