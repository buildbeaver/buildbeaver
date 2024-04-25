package runner

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/fatih/structs"

	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/api/rest/documents"
)

type jobTemplateData struct {
	Name        string `structs:"name"`
	Fingerprint string `structs:"fingerprint"`
}

// fieldTemplateRegex matches our standard template syntax of "${{ job_name.nested.value }}
var fieldTemplateRegex = regexp.MustCompile("\\$\\{\\{ *(.+?) *}}")

// templateJob applies template substitution to every support field of the RunnableJob in place.
func templateJob(runnable *documents.RunnableJob) error {
	// NOTE in time we will make more variables available for templating, so we've nested
	// jobs under a top-level "jobs" key. You can imagine we may end up with other top-level
	// keys like "job" (self referencing), "vars" (global config) etc.
	rootContext := make(map[string]interface{})
	jobContext := make(map[string]interface{})
	for _, job := range runnable.Jobs {
		jobData := &jobTemplateData{
			Name:        job.Name.String(),
			Fingerprint: job.Fingerprint,
		}
		jobContext[job.Name.String()] = structs.Map(jobData)
	}
	rootContext["jobs"] = jobContext
	if runnable.Job.DockerConfig != nil {
		v, err := templateField(runnable.Job.DockerConfig.Image, rootContext)
		if err != nil {
			return fmt.Errorf("error templating docker image field: %w", err)
		}
		runnable.Job.DockerConfig.Image = v
	}
	return nil
}

// templateField substitutes all templates in the specified field value (if any) with corresponding
// variables from the data map.
func templateField(value string, templateDataByJobName map[string]interface{}) (string, error) {
	matches := fieldTemplateRegex.FindAllStringSubmatch(value, 256) // Some upper bound we expect to never be hit
	if len(matches) == 0 {
		return value, nil
	}
	for _, match := range matches {
		if len(match) != 2 {
			return "", fmt.Errorf("error unexpected regex result (!=2)")
		}
		outer := match[0]                  // e.g. "${{ foo.bar }}
		inner := match[1]                  // e.g. "foo.bar"
		parts := strings.Split(inner, ".") // e.g. ["foo", "bar"]
		for _, part := range parts {
			// Each part (including the variable names that are template in) must be valid names
			if !models.ResourceNameRegex.Match([]byte(part)) {
				return "", fmt.Errorf("error invalid path part: %s", part)
			}
			var current interface{} = templateDataByJobName
			for _, part := range parts {
				currentM, ok := current.(map[string]interface{})
				if !ok {
					return "", fmt.Errorf("error unknown %q", match)
				}
				next, ok := currentM[part]
				if !ok {
					return "", fmt.Errorf("error resolving %q", match)
				}
				current = next
			}
			switch current.(type) {
			case string, int, int32, int64, float32, float64, bool:
			default:
				return "", fmt.Errorf("error only primitive types can be templated")
			}
			value = strings.Replace(value, outer, fmt.Sprintf("%v", current), 1)
		}
	}
	return value, nil
}
