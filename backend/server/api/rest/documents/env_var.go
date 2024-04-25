package documents

import "github.com/buildbeaver/buildbeaver/common/models"

// EnvVar represents a single key/value pair to export as an
// environment variable prior to executing a step.
type EnvVar struct {
	// Name of the environment variable
	Name string `json:"name"`
	// Value of the string variable, if the variable is set explicitly.
	Value string `json:"value"`
	// ValueFromSecret is the name of the secret to set this
	// variable to, if setting the variable to a secret.
	ValueFromSecret string `json:"value_from_secret"`
}

func MakeEnvVar(envVar *models.EnvVar) *EnvVar {
	return &EnvVar{
		Name:            envVar.Name,
		Value:           envVar.Value,
		ValueFromSecret: envVar.ValueFromSecret,
	}
}

func MakeEnvVars(environment []*models.EnvVar) []*EnvVar {
	var docs []*EnvVar
	for _, envVar := range environment {
		docs = append(docs, MakeEnvVar(envVar))
	}
	return docs
}
