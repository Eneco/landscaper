package landscaper

import (
	"github.com/Sirupsen/logrus"
	"k8s.io/client-go/1.4/pkg/api/v1"
)

// Secrets is currently a slice of secret names that should be applied to a component
type Secrets map[string]string

// SecretsProvider reads secrets for a release from both the desired state as well as the current state
type SecretsProvider interface {
	Read(string) (Secrets, error)
	Write(string, Secrets) error
}

type secretsProvider struct {
	env *Environment
}

// NewSecretsProvider is a factory method to create a new SecretsProvider
func NewSecretsProvider(env *Environment) SecretsProvider {
	return &secretsProvider{env: env}
}

func (sp *secretsProvider) Read(componentName string) (Secrets, error) {
	logrus.WithField("component", componentName).Info("Reading secrets for component")

	secret, err := sp.env.KubeClient().Secrets(sp.env.Namespace).Get(componentName)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"component": componentName,
			"error":     err,
		}).Error("Error when reading secrets for component")
		return nil, err
	}

	logrus.WithField("component", componentName).Info("Successfully read secrets for component")

	secrets := Secrets{}
	for key, val := range secret.Data {
		secrets[key] = string(val)
	}

	return secrets, nil
}

func (sp *secretsProvider) Write(componentName string, secrets Secrets) error {
	logrus.WithField("component", componentName).Info("Writing secrets for component")

	_, err := sp.env.KubeClient().Secrets(sp.env.Namespace).Create(&v1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name: componentName,
		},
		StringData: secrets,
	})
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"component": componentName,
			"error":     err,
		}).Error("Error when writing secrets for component")
		return err
	}

	logrus.WithField("component", componentName).Info("Successfully written secrets for component")

	return nil
}
