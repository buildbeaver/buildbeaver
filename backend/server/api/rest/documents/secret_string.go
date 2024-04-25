package documents

// SecretString provides a way of adding our standard "retrieve from secret" fields to a struct.
type SecretString struct {
	// Value of the string variable, if the variable is set explicitly.
	Value string `json:"value"`
	// FromSecret is the name of the secret to set this
	// variable to, if setting the variable to a secret.
	FromSecret string `json:"value_from_secret"`
}
