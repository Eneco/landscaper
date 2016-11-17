package landscaper

// Secrets is currently a slice of secret names that should be applied to a component
type Secrets map[string]interface{}

// SecretsProvider reads secrets for a release from both the desired state as well as the current state
type SecretsProvider interface {
	Read(string) (Secrets, error)
	Write(string, Secrets) error
}

type secretsProvider struct {
	env *Environment
}

// NewSecretsProvider is a factory method to create a new SecretsProvider
func NewSecretsProvider(env *Environment) (SecretsProvider, error) {
	err := env.EnsureKubeClient()
	if err != nil {
		return nil, err
	}

	return &secretsProvider{env: env}, nil
}

func (sp *secretsProvider) Read(releaseName string) (Secrets, error) {
	return Secrets{}, nil
}

func (sp *secretsProvider) Write(releaseName string, secrets Secrets) error {
	return nil
}
