package landscaper

import (
	"os"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/fatih/camelcase"
	"k8s.io/client-go/1.4/pkg/api/v1"
)

// Secrets is currently a slice of secret names that should be applied to a component
type Secrets []string

// SecretValues is a map containing the actual values of the secrets. Note that this should not be written
// to kubernetes or anywhere else persistent!
type SecretValues map[string]string

// SecretsProvider reads secrets for a release from both the desired state as well as the current state
type SecretsProvider interface {
	Read(componentName string) (SecretValues, error)
	Write(componentName string, secretValues SecretValues, isUpdate bool) error
	Delete(componentName string) error
}

type secretsProvider struct {
	env *Environment
}

// NewSecretsProvider is a factory method to create a new SecretsProvider
func NewSecretsProvider(env *Environment) SecretsProvider {
	return &secretsProvider{env: env}
}

func (sp *secretsProvider) Read(componentName string) (SecretValues, error) {
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

	secrets := SecretValues{}
	for key, val := range secret.Data {
		secrets[key] = string(val)
	}

	return secrets, nil
}

func (sp *secretsProvider) Write(componentName string, secrets SecretValues, isUpdate bool) error {
	logrus.WithField("component", componentName).Info("Writing secrets for component")

	if isUpdate {
		err := sp.Delete(componentName)
		if err != nil {
			return err
		}
	}

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

func (sp *secretsProvider) Delete(componentName string) error {
	logrus.WithField("component", componentName).Error("Deleting existing secrets for component")

	// We first completely delete the current secrets
	err := sp.env.KubeClient().Secrets(sp.env.Namespace).Delete(componentName, nil)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"component": componentName,
			"error":     err,
		}).Error("Error when deleting current secrets for component")
		return err
	}

	return nil
}

func readSecretValues(cmp *Component) {
	for _, key := range cmp.Secrets {
		parts := camelcase.Split(key)
		for i, val := range parts {
			parts[i] = strings.ToUpper(val)
		}
		envName := strings.Join(parts, "_")

		secretValue := os.Getenv(envName)
		if len(secretValue) == 0 {
			logrus.WithFields(logrus.Fields{"secret": key, "envName": envName}).Warn("Secret not found in environment")
		}

		cmp.SecretValues[key] = secretValue
	}
}
