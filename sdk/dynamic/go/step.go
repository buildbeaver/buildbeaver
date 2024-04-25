package bb

import (
	"github.com/buildbeaver/sdk/dynamic/bb/client"
)

type Step struct {
	definition client.StepDefinition
}

func NewStep() *Step {
	return &Step{definition: client.StepDefinition{}}
}

func (step *Step) GetData() client.StepDefinition {
	return step.definition
}

func (step *Step) GetName() ResourceName {
	return ResourceName(step.definition.Name)
}

func (step *Step) Name(name string) *Step {
	step.definition.Name = name
	return step
}

func (step *Step) Desc(description string) *Step {
	step.definition.Description = &description
	return step
}

func (step *Step) Commands(commands ...string) *Step {
	step.definition.Commands = append(step.definition.Commands, commands...)
	return step
}

func (step *Step) Depends(stepNames ...string) *Step {
	step.definition.Depends = append(step.definition.Depends, stepNames...)
	return step
}

func (step *Step) DependsOnSteps(steps ...*Step) *Step {
	for _, dependsOnStep := range steps {
		step.definition.Depends = append(step.definition.Depends, dependsOnStep.definition.Name)
	}
	return step
}
