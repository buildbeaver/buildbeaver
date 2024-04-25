package bb

// SecretName is the name of a secret used to provide a value for environment variables or authentication.
type SecretName string

func (s SecretName) String() string {
	return string(s)
}
