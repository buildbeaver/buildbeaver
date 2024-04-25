package models

import (
	"database/sql/driver"
	"fmt"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

// NodeFQN is the Fully Qualified Name identifying a node in the build graph.
type NodeFQN struct {
	WorkflowName ResourceName `json:"workflow_name"`
	JobName      ResourceName `json:"job_name"`
	StepName     ResourceName `json:"step_name"`
}

func NewNodeFQN(workflowName ResourceName, jobName ResourceName, stepName ResourceName) NodeFQN {
	return NodeFQN{
		WorkflowName: workflowName,
		JobName:      jobName,
		StepName:     stepName,
	}
}

func NewNodeFQNForJob(workflowName ResourceName, jobName ResourceName) NodeFQN {
	return NodeFQN{
		WorkflowName: workflowName,
		JobName:      jobName,
		StepName:     "",
	}
}

func NewNodeFQNForWorkflow(workflowName ResourceName) NodeFQN {
	return NodeFQN{
		WorkflowName: workflowName,
		JobName:      "",
		StepName:     "",
	}
}

func (s *NodeFQN) String() string {
	if s.JobName == "" && s.StepName == "" {
		return s.WorkflowName.String()
	} else if s.StepName == "" {
		return fmt.Sprintf("%s.%s", s.WorkflowName, s.JobName)
	} else {
		return fmt.Sprintf("%s.%s.%s", s.WorkflowName, s.JobName, s.StepName)
	}
}

func (s *NodeFQN) Equal(that *NodeFQN) bool {
	return s.String() == that.String()
}

func (s *NodeFQN) Scan(src interface{}) error {
	str, ok := src.(string)
	if !ok {
		return fmt.Errorf("error expected step FQN to be string, got: %#v", src)
	}
	parts := strings.SplitN(str, ".", 3)
	if len(parts) == 1 {
		s.WorkflowName = ResourceName(parts[0])
		s.JobName = ""
		s.StepName = ""
	} else if len(parts) == 2 {
		s.WorkflowName = ResourceName(parts[0])
		s.JobName = ResourceName(parts[1])
		s.StepName = ""
	} else if len(parts) == 3 {
		s.WorkflowName = ResourceName(parts[0])
		s.JobName = ResourceName(parts[1])
		s.StepName = ResourceName(parts[2])
	} else {
		return fmt.Errorf("error expected one, two or three parts to Node FQN in the format \"workflow.job.step\" but found %q", str)
	}
	return nil
}

func (s *NodeFQN) ScanWorkflowOnly(src interface{}) error {
	str, ok := src.(string)
	if !ok {
		return fmt.Errorf("error expected FQN to be string, got: %#v", src)
	}
	parts := strings.SplitN(str, ".", 3)
	if len(parts) == 1 {
		s.WorkflowName = ResourceName(parts[0])
		s.JobName = ""
		s.StepName = ""
	} else {
		return fmt.Errorf("error expected only workflow name in Node Fully Qualified Name, but found %q", str)
	}
	return nil
}

func (s *NodeFQN) Value() (driver.Value, error) {
	return s.String(), nil
}

func (s *NodeFQN) Validate() error {
	var result *multierror.Error
	if s.WorkflowName == "" {
		result = multierror.Append(result, errors.New("Workflow name must be specified"))
	} else if !ResourceNameRegex.MatchString(s.WorkflowName.String()) {
		result = multierror.Append(result, errors.New("Workflow name must only contain alphanumeric, dash or underscore characters (matching ^[a-zA-Z0-9_-]+$)"))
	}
	if s.JobName == "" && s.StepName != "" {
		result = multierror.Append(result, errors.New("Job name must be specified if a step name is specified"))
	}
	if s.JobName != "" && !ResourceNameRegex.MatchString(s.JobName.String()) {
		result = multierror.Append(result, errors.New("Job name must only contain alphanumeric, dash or underscore characters (matching ^[a-zA-Z0-9_-]+$)"))
	}
	if s.StepName != "" && !ResourceNameRegex.MatchString(s.StepName.String()) {
		result = multierror.Append(result, errors.New("Step name must only contain alphanumeric, dash or underscore characters (matching ^[a-zA-Z0-9_-]+$)"))
	}
	return result.ErrorOrNil()
}
