package parser

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/buildbeaver/buildbeaver/common/gerror"
	"github.com/pkg/errors"

	"github.com/buildbeaver/buildbeaver/common/models"
)

var (
	jobDependsOnOneArtifactFromJobRegex02           = regexp.MustCompile(`(?im)^jobs\.([a-zA-Z0-9_*-]+)\.artifacts\.([a-zA-Z0-9_-]+)$`)
	jobDependsOnAllArtifactsFromJobRegex02          = regexp.MustCompile(`(?im)^jobs\.([a-zA-Z0-9_*-]+)\.artifacts$`)
	jobDependsOnAllArtifactsFromJobShorthandRegex02 = regexp.MustCompile(`(?im)^([a-zA-Z0-9_*-]+)\.artifacts$`)
	jobDependsOnJobRegex02                          = regexp.MustCompile(`(?im)^jobs\.([a-zA-Z0-9_*-]+)$`)
	jobDependsOnJobShorthandRegex02                 = regexp.MustCompile(`(?im)^([a-zA-Z0-9_*-]+)$`)
)

type buildDefinitionParserV02 struct {
	limits ParserLimits
}

func newBuildDefinitionParserV02(limits ParserLimits) *buildDefinitionParserV02 {
	return &buildDefinitionParserV02{
		limits: limits,
	}
}

// Parse parses a build definition of this specific version.
func (s *buildDefinitionParserV02) Parse(topLevelElement map[string]interface{}) (*models.BuildDefinition, error) {
	rJobs, ok := topLevelElement["jobs"]
	if !ok {
		return nil, errors.Errorf("build definition does not contain a 'jobs' list")
	}
	rJobsArray, ok := rJobs.([]interface{})
	if !ok {
		return nil, errors.Errorf("job element must contain an array but found %T", rJobs)
	}
	// Call the parseJobs function for version 1.0
	// This entire set of functions will need to be maintained and possibly duplicated for the next version.
	jobs, err := s.parseJobs(rJobsArray)
	if err != nil {
		return nil, err
	}
	build := &models.BuildDefinition{Jobs: jobs}
	return build, nil
}

func (s *buildDefinitionParserV02) parseJobs(raw []interface{}) ([]models.JobDefinition, error) {
	jobs := make([]models.JobDefinition, len(raw))
	for i, obj := range raw {
		element, ok := obj.(map[string]interface{})
		if !ok {
			return nil, errors.Errorf("Top-level element is not a job object: %T", obj)
		}
		kind, ok := element["kind"]
		if !ok || kind == "pipeline" || kind == "job" {
			job, err := s.parseJob(element)
			if err != nil {
				return nil, errors.Wrapf(err, "error parsing pipeline job at index %d", i)
			}
			jobs[i] = *job
		} else {
			return nil, errors.Errorf("Unsupported kind: %s", kind)
		}
	}
	return jobs, nil
}

func (s *buildDefinitionParserV02) parseJob(raw map[string]interface{}) (*models.JobDefinition, error) {

	job := &models.JobDefinition{}

	rName, ok := raw["name"]
	if ok {
		name, ok := rName.(string)
		if !ok {
			return nil, errors.Errorf("Expected job 'name' field to be a string but found: %T", rName)
		}
		job.Name = models.ResourceName(name)
	}

	rDescription, ok := raw["description"]
	if ok {
		job.Description, ok = rDescription.(string)
		if !ok {
			return nil, errors.Errorf("Expected job 'description' field to be a string but found: %T", rDescription)
		}
	}

	rRunsOn, ok := raw["runs_on"]
	if ok {
		switch value := rRunsOn.(type) {
		case string:
			job.RunsOn = []models.Label{models.Label(value)}
		case []interface{}:
			labels, err := s.parseLabels(value)
			if err != nil {
				return nil, errors.Wrap(err, "Unable to parse job 'runs-on' field")
			}
			job.RunsOn = labels
		default:
			return nil, errors.Errorf("Unable to parse %q to list of labels", rRunsOn)
		}
	}

	rDepends, ok := raw["depends"]
	if ok {
		jobDependencies, err := s.parseJobDependencies(rDepends)
		if err != nil {
			return nil, errors.Wrap(err, "error parsing job dependencies")
		}
		job.Depends = jobDependencies
	}

	// If type is not set explicitly then we will try and infer it below
	rType, ok := raw["type"]
	if ok {
		err := job.Type.Scan(rType)
		if err != nil {
			return nil, fmt.Errorf("error parsing job 'type' property: %w", err)
		}
	}

	rDocker, ok := raw["docker"]
	if ok {
		if job.Type.Valid() && job.Type != models.JobTypeDocker {
			return nil, fmt.Errorf("%s jobs do not support a 'docker' configuration option", job.Type)
		}
		job.Type = models.JobTypeDocker

		docker, ok := rDocker.(map[string]interface{})
		if !ok {
			return nil, errors.Errorf("Expected job 'docker' field to be an object but found: %T", rDocker)
		}

		rShell, ok := docker["shell"]
		if ok {
			if shell, ok := rShell.(string); ok {
				job.DockerShell = &shell
			} else {
				return nil, errors.Errorf("Expected job 'docker.shell' field to be a string but found: %T", rShell)
			}
		}

		rImage, ok := docker["image"]
		if ok {
			job.DockerImage, ok = rImage.(string)
			if !ok {
				return nil, errors.Errorf("Expected job 'docker.image' field to be a string but found: %T", rImage)
			}
		}

		rPull := docker["pull"]
		err := job.DockerImagePullStrategy.Scan(rPull) // handles the default case if pull is not set
		if err != nil {
			return nil, fmt.Errorf("error parsing job 'docker.pull' property: %w", err)
		}

		auth, err := s.parseDockerAuthOrNil(docker)
		if err != nil {
			return nil, err
		}
		job.DockerAuth = auth
	}

	rStepExecution := raw["step_execution"]
	err := job.StepExecution.Scan(rStepExecution)
	if err != nil {
		return nil, fmt.Errorf("error parsing job 'step_execution' property: %w", err)
	}

	rServices, ok := raw["services"]
	if ok {
		value, ok := rServices.([]interface{})
		if !ok {
			return nil, errors.Errorf("Expected services to be an array of service objects but found %T", rServices)
		}
		for i, obj := range value {
			element, ok := obj.(map[string]interface{})
			if !ok {
				return nil, errors.Errorf("Expected services to be an array of service objects but found %T", obj)
			}
			service, err := s.parseService(element)
			if err != nil {
				return nil, errors.Wrapf(err, "Error parsing service at index %d", i)
			}
			job.Services = append(job.Services, service)
		}
	}

	rFingerprintCommands, ok := raw["fingerprint"]
	if ok {
		switch value := rFingerprintCommands.(type) {
		case string:
			job.FingerprintCommands = []models.Command{models.Command(value)}
		case []interface{}:
			fingerprintCommands, err := s.parseCommands(value)
			if err != nil {
				return nil, errors.Wrap(err, "Unable to parse job 'fingerprint' field")
			}
			job.FingerprintCommands = fingerprintCommands
		default:
			return nil, errors.Errorf("Unable to parse %q to list of fingerprint commands", rFingerprintCommands)
		}
	}

	rArtifacts, ok := raw["artifacts"]
	if ok {

		rValues, ok := rArtifacts.([]interface{})
		if !ok {
			return nil, errors.Errorf("Unable to parse %q to list of artifacts", rArtifacts)
		}

		// Artifact definitions can be an array of objects or an array of strings. Objects are used when
		// additional information is attached to the artifact (like a name) and strings are used when the
		// default name is used.
		if len(rValues) > 0 {
			switch rValues[0].(type) {
			case string:
				artifacts, err := s.parseArtifactDefinitionStrings(rValues)
				if err != nil {
					return nil, errors.Wrap(err, "error parsing artifact dependencies")
				}
				job.ArtifactDefinitions = artifacts

			case interface{}:
				artifacts, err := s.parseArtifactDefinitionObjects(rValues)
				if err != nil {
					return nil, errors.Wrap(err, "error parsing artifact dependencies")
				}
				job.ArtifactDefinitions = artifacts

			default:
				return nil, errors.Errorf("Expected 'artifacts' to contain an array of strings referencing other " +
					"step's artifacts as dependencies, or a map of objects describing the artifacts this step will produce")
			}
		}
	}

	rSteps, ok := raw["steps"]
	if ok {
		value, ok := rSteps.([]interface{})
		if !ok {
			return nil, errors.Errorf("Expected steps to be an array of step objects but found %T", rSteps)
		}
		for i, obj := range value {
			element, ok := obj.(map[string]interface{})
			if !ok {
				return nil, errors.Errorf("Expected steps to be an array of step objects but found %T", obj)
			}
			step, err := s.parseStep(job, element)
			if err != nil {
				return nil, errors.Wrapf(err, "Error parsing step at index %d", i)
			}
			job.Steps = append(job.Steps, *step)
			if s.limits.MaxStepsPerJob > 0 && len(job.Steps) > s.limits.MaxStepsPerJob {
				return nil, gerror.NewErrValidationFailed(
					fmt.Sprintf("too many steps in job '%s'; a maximum of %d steps are allowed in each job",
						job.Name, s.limits.MaxStepsPerJob))
			}
		}
	}

	return job, nil
}

func (s *buildDefinitionParserV02) parseStep(job *models.JobDefinition, raw map[string]interface{}) (*models.StepDefinition, error) {

	step := &models.StepDefinition{}

	rName, ok := raw["name"]
	if ok {
		name, ok := rName.(string)
		if !ok {
			return nil, errors.Errorf("Expected step 'name' field to be a string but found: %T", rName)
		}
		step.Name = models.ResourceName(name)
	}

	rDescription, ok := raw["description"]
	if ok {
		step.Description, ok = rDescription.(string)
		if !ok {
			return nil, errors.Errorf("Expected step 'description' field to be a string but found: %T", rDescription)
		}
	}

	rCommands, ok := raw["commands"]
	if ok {
		switch value := rCommands.(type) {
		case string:
			step.Commands = []models.Command{models.Command(value)}
		case []interface{}:
			commands, err := s.parseCommands(value)
			if err != nil {
				return nil, errors.Wrap(err, "Unable to parse step 'commands' field")
			}
			step.Commands = commands
		default:
			return nil, errors.Errorf("Unable to parse %q to list of commands", rCommands)
		}
	}

	depends, err := s.parseStepDependencies(job, raw)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing step dependencies")
	}
	step.Depends = depends

	rEnvironment, ok := raw["environment"]
	if ok {
		environment, err := s.parseEnvironment(rEnvironment)
		if err != nil {
			return nil, err
		}
		job.Environment = job.Environment.Merge(environment)
	}

	return step, nil
}

// parseJobDependencies parses a jobs's dependencies to a structured list of other jobs and job artifacts
// that this job depends on.
func (s *buildDefinitionParserV02) parseJobDependencies(raw interface{}) ([]*models.JobDependency, error) {
	var rawArr []interface{}
	switch value := raw.(type) {
	case string:
		rawArr = []interface{}{value}
	case []interface{}:
		rawArr = value
	default:
		return nil, errors.Errorf("unable to parse %q to a list of job dependencies", value)
	}

	jobDependenciesByJobName := map[models.ResourceName]*models.JobDependency{}
	recordJobDependency := func(jobName models.ResourceName, artifactDeps ...*models.ArtifactDependency) {
		if existing, ok := jobDependenciesByJobName[jobName]; ok {
			if len(artifactDeps) > 0 {
				existing.ArtifactDependencies = append(existing.ArtifactDependencies, artifactDeps...)
			}
		} else {
			jobDependenciesByJobName[jobName] = models.NewJobDependency("", jobName, artifactDeps...)
		}
	}

	for _, rValue := range rawArr {
		value, ok := rValue.(string)
		if !ok {
			return nil, errors.Errorf("unable to parse %q type %T to a job dependency", rValue, rValue)
		}

		match := jobDependsOnOneArtifactFromJobRegex02.FindStringSubmatch(value)
		if match != nil {
			jobName := models.ResourceName(match[1])
			artifactName := models.ResourceName(match[2])
			artifactDep := models.NewArtifactDependency("", jobName, artifactName)
			recordJobDependency(jobName, artifactDep)
			continue
		}

		match = jobDependsOnAllArtifactsFromJobRegex02.FindStringSubmatch(value)
		if match != nil {
			jobName := models.ResourceName(match[1])
			artifactDep := models.NewArtifactDependency("", jobName, "")
			recordJobDependency(jobName, artifactDep)
			continue
		}

		match = jobDependsOnAllArtifactsFromJobShorthandRegex02.FindStringSubmatch(value)
		if match != nil {
			jobName := models.ResourceName(match[1])
			artifactDep := models.NewArtifactDependency("", jobName, "")
			recordJobDependency(jobName, artifactDep)
			continue
		}

		match = jobDependsOnJobRegex02.FindStringSubmatch(value)
		if match != nil {
			jobName := models.ResourceName(match[1])
			recordJobDependency(jobName)
			continue
		}

		match = jobDependsOnJobShorthandRegex02.FindStringSubmatch(value)
		if match != nil {
			jobName := models.ResourceName(match[1])
			recordJobDependency(jobName)
			continue
		}

		return nil, errors.Errorf("Unable to parse %q to a step dependency", rValue)
	}

	jobDependencies := make([]*models.JobDependency, 0, len(jobDependenciesByJobName))
	for _, dep := range jobDependenciesByJobName {
		jobDependencies = append(jobDependencies, dep)
	}
	return jobDependencies, nil
}

func (s *buildDefinitionParserV02) parseStepDependencies(job *models.JobDefinition, raw map[string]interface{}) ([]*models.StepDependency, error) {
	var rawArr []interface{}
	rDepends, ok := raw["depends"]
	if ok {
		switch value := rDepends.(type) {
		case string:
			rawArr = []interface{}{value}
		case []interface{}:
			rawArr = value
		default:
			return nil, errors.Errorf("Unable to parse %q to a list of step dependencies", value)
		}
	}
	switch job.StepExecution {
	case models.StepExecutionSequential:
		if len(rawArr) > 0 {
			return nil, fmt.Errorf("steps may only use 'depends' when the job is configured to use parallel step execution")
		}
		if len(job.Steps) > 0 { // Add a dependency between steps to preserve ordering for sequential execution
			return []*models.StepDependency{models.NewStepDependency(job.Steps[len(job.Steps)-1].Name)}, nil
		}
		return nil, nil
	case models.StepExecutionParallel:
		stepDependenciesByStepName := map[models.ResourceName]*models.StepDependency{}
		for _, rValue := range rawArr {
			switch value := rValue.(type) {
			case string:
				name := models.ResourceName(value)
				if _, ok := stepDependenciesByStepName[name]; !ok {
					stepDependenciesByStepName[name] = models.NewStepDependency(name)
				}
			default:
				return nil, errors.Errorf("Unable to parse %q to a step dependency", rValue)
			}
		}
		stepDependencies := make([]*models.StepDependency, 0, len(stepDependenciesByStepName))
		for _, dep := range stepDependenciesByStepName {
			stepDependencies = append(stepDependencies, dep)
		}
		return stepDependencies, nil
	default:
		panic("unknown step execution")
	}
}

func (s *buildDefinitionParserV02) parseArtifactDefinitionStrings(raw []interface{}) ([]*models.ArtifactDefinition, error) {
	// If artifacts are defined as an array of strings then those strings represent
	// the paths and the artifact(s) take on a default name.
	artifact := &models.ArtifactDefinition{
		GroupName: "default",
	}
	for _, rValue := range raw {
		switch value := rValue.(type) {
		case string:
			artifact.Paths = append(artifact.Paths, value)
		default:
			return nil, errors.Errorf("Unable to parse %q to an artifact path", rValue)
		}
	}
	return []*models.ArtifactDefinition{artifact}, nil
}

func (s *buildDefinitionParserV02) parseArtifactDefinitionObjects(raw []interface{}) ([]*models.ArtifactDefinition, error) {
	var artifacts []*models.ArtifactDefinition
	for _, rValue := range raw {
		switch value := rValue.(type) {
		case map[string]interface{}:
			definition := &models.ArtifactDefinition{}
			rName, ok := value["name"]
			if ok {
				name, ok := rName.(string)
				if !ok {
					return nil, errors.Errorf("Expected artifact definition 'name' field to be a string but found: %T", rName)
				}
				definition.GroupName = models.ResourceName(name)
			}
			rPath, ok := value["paths"]
			if ok {
				switch value := rPath.(type) {
				case string:
					definition.Paths = []string{value}
				case []interface{}:
					paths, err := s.parseStringArray(value)
					if err != nil {
						return nil, errors.Wrap(err, "Unable to parse artifact 'paths' field")
					}
					definition.Paths = paths
				default:
					return nil, errors.Errorf("Unable to parse %q to list of artifact paths", rPath)
				}
			}
			artifacts = append(artifacts, definition)
		default:
			return nil, errors.Errorf("Unable to parse %q to an artifact definition", rValue)
		}
	}
	return artifacts, nil
}

func (s *buildDefinitionParserV02) parseService(raw map[string]interface{}) (*models.Service, error) {
	service := &models.Service{}
	rName, ok := raw["name"]
	if ok {
		service.Name, ok = rName.(string)
		if !ok {
			return nil, errors.Errorf("Expected service 'name' field to be a string but found: %T", rName)
		}
	}
	rImage, ok := raw["image"]
	if ok {
		service.DockerImage, ok = rImage.(string)
		if !ok {
			return nil, errors.Errorf("Expected service 'image' field to be a string but found: %T", rImage)
		}
	}
	auth, err := s.parseDockerAuthOrNil(raw)
	if err != nil {
		return nil, err
	}
	service.DockerRegistryAuthentication = auth
	rEnvironment, ok := raw["environment"]
	if ok {
		environment, err := s.parseEnvironment(rEnvironment)
		if err != nil {
			return nil, err
		}
		service.Environment = environment
	}
	return service, nil
}

func (s *buildDefinitionParserV02) parseEnvironment(raw interface{}) ([]*models.EnvVar, error) {
	var environment []*models.EnvVar
	rValues, ok := raw.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("error expected 'environment' field to contain map of key/value pairs")
	}
	for key, rValue := range rValues {
		secretString, err := s.parseSecretString(rValue)
		if err != nil {
			return nil, fmt.Errorf("error parsing value for environment variable %q: %w", key, err)
		}
		environment = append(environment, &models.EnvVar{Name: key, SecretString: *secretString})
	}
	return environment, nil
}

func (s *buildDefinitionParserV02) parseCommands(raw []interface{}) (models.Commands, error) {
	var commands models.Commands
	for _, rawStr := range raw {
		switch value := rawStr.(type) {
		case string:
			commands = append(commands, models.Command(value))
		default:
			return nil, errors.Errorf("Unable to parse %q to a command", rawStr)
		}
	}
	return commands, nil
}

func (s *buildDefinitionParserV02) parseLabels(raw []interface{}) (models.Labels, error) {
	var labels models.Labels
	for _, rawStr := range raw {
		switch value := rawStr.(type) {
		case string:
			labels = append(labels, models.Label(value))
		default:
			return nil, errors.Errorf("Unable to parse %q to a label", rawStr)
		}
	}
	return labels, nil
}

// parseStringArray attempts to convert an []interface{} to an array of strings.
// Supports parsing of string, int and bool values. If the interface{} value is not
// one of these then an error is returned.
func (s *buildDefinitionParserV02) parseStringArray(raw []interface{}) ([]string, error) {
	var strs []string
	for _, rawStr := range raw {
		switch value := rawStr.(type) {
		case string:
			strs = append(strs, value)
		case int:
			strs = append(strs, strconv.FormatInt(int64(value), 10))
		case bool:
			strs = append(strs, strconv.FormatBool(value))
		default:
			return nil, errors.Errorf("Unable to parse %q to a string", rawStr)
		}
	}
	return strs, nil
}

// parseSecretString attempts to convert the raw value of a field into a SecretString. The raw value can contain
// either a string (a literal value) or an object. If an object is provided then it can contain either a
// literal value or the name of a secret that will contain the value. An empty string is a valid literal value.
// Returns an error if any unexpected data is provided.
func (s *buildDefinitionParserV02) parseSecretString(rValue interface{}) (*models.SecretString, error) {
	switch value := rValue.(type) {
	case string:
		return &models.SecretString{Value: value}, nil
	case map[string]interface{}:
		// A value object must contain either a 'value' key or a 'from_secret' key, but not both
		rFromSecret, foundSecret := value["from_secret"]
		rActualValue, foundValue := value["value"]
		if foundSecret && foundValue {
			return nil, errors.Errorf("Expected value object to contain either a 'from_secret' field or a 'value' field but found both")
		}

		if foundSecret {
			fromSecret, ok := rFromSecret.(string)
			if !ok {
				return nil, errors.Errorf("Expected 'from_secret' field to be a string but found: %T", fromSecret)
			}
			return &models.SecretString{ValueFromSecret: fromSecret}, nil
		}

		if foundValue {
			actualValue, ok := rActualValue.(string)
			if !ok {
				return nil, errors.Errorf("Expected 'value' field to be a string but found: %T", actualValue)
			}
			return &models.SecretString{Value: actualValue}, nil
		}
	}
	return nil, errors.Errorf("Unable to parse %q to a value (either literal or secret)", rValue)
}

func (s *buildDefinitionParserV02) parseDockerAuthOrNil(docker map[string]interface{}) (*models.DockerAuth, error) {

	var auth *models.DockerAuth

	rBasicAuth, ok := docker["basic_auth"]
	if ok {
		value, ok := rBasicAuth.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("error expected Docker basic auth to be an object but found %T", rBasicAuth)
		}
		if auth == nil {
			auth = &models.DockerAuth{}
		}
		auth.Basic = &models.DockerBasicAuth{}
		rUsername, ok := value["username"]
		if ok {
			secretString, err := s.parseSecretString(rUsername)
			if err != nil {
				return nil, fmt.Errorf("error parsing Docker basic auth username: %w", err)
			}
			auth.Basic.Username = *secretString
		}
		rPassword, ok := value["password"]
		if ok {
			secretString, err := s.parseSecretString(rPassword)
			if err != nil {
				return nil, fmt.Errorf("error parsing Docker basic auth password: %w", err)
			}
			if secretString.ValueFromSecret == "" || secretString.Value != "" {
				return nil, fmt.Errorf("error Docker basic auth password must be configured to use a secret: %w", err)
			}
			auth.Basic.Password = *secretString
		}
		if auth.Basic.Username.Value == "" && auth.Basic.Username.ValueFromSecret == "" {
			return nil, fmt.Errorf("error username must be set when using Docker basic auth")
		}
		if auth.Basic.Password.Value == "" && auth.Basic.Password.ValueFromSecret == "" {
			return nil, fmt.Errorf("error password must be set when using Docker basic auth")
		}
	}

	rAWSAuth, ok := docker["aws_auth"]
	if ok {
		value, ok := rAWSAuth.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("error expected Docker AWS auth to be an object but found %T", rAWSAuth)
		}
		if auth == nil {
			auth = &models.DockerAuth{}
		}
		auth.AWS = &models.DockerAWSAuth{}
		rRegion, ok := value["aws_region"]
		if ok {
			value, ok := rRegion.(string)
			if !ok {
				return nil, fmt.Errorf("error expected Docker AWS auth region to be a string but found: %T", rRegion)
			}
			auth.AWS.AWSRegion = value
		}
		rUsername, ok := value["aws_access_key_id"]
		if ok {
			secretString, err := s.parseSecretString(rUsername)
			if err != nil {
				return nil, fmt.Errorf("error parsing Docker AWS auth aws_access_key_id: %w", err)
			}
			auth.AWS.AWSAccessKeyID = *secretString
		}
		rPassword, ok := value["aws_secret_access_key"]
		if ok {
			secretString, err := s.parseSecretString(rPassword)
			if err != nil {
				return nil, fmt.Errorf("error parsing Docker AWS auth aws_secret_access_key: %w", err)
			}
			if secretString.ValueFromSecret == "" || secretString.Value != "" {
				return nil, fmt.Errorf("error Docker AWS auth aws_secret_access_key must be configured to use a secret: %w", err)
			}
			auth.AWS.AWSSecretAccessKey = *secretString
		}
		if auth.AWS.AWSAccessKeyID.Value == "" && auth.AWS.AWSAccessKeyID.ValueFromSecret == "" {
			return nil, fmt.Errorf("error aws_access_key_id must be set when using Docker AWS auth")
		}
		if auth.AWS.AWSSecretAccessKey.Value == "" && auth.AWS.AWSSecretAccessKey.ValueFromSecret == "" {
			return nil, fmt.Errorf("error aws_secret_access_key must be set when using Docker AWS auth")
		}
	}

	return auth, nil
}

func (s *buildDefinitionParserV02) dedupeStringSlice(list []string) []string {
	// This is not optimal but, ¯\_(ツ)_/¯
	unique := make(map[string]bool)
	for _, str := range list {
		unique[str] = true
	}
	var ret []string
	for k := range unique {
		ret = append(ret, k)
	}
	return ret
}
