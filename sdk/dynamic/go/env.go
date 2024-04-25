package bb

type Env struct {
	name       string
	value      string
	secretName string
}

func NewEnv() *Env { return &Env{} }

func (m *Env) Name(name string) *Env {
	m.name = name
	return m
}

func (m *Env) Value(value string) *Env {
	m.value = value
	return m
}

func (m *Env) ValueFromSecret(secretName string) *Env {
	m.secretName = secretName
	return m
}
