package landscaper

import (
	"os"
	"strings"

	"github.com/Sirupsen/logrus"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/errors"
)

// Secrets is currently a slice of secret names that should be applied to a component
type Secrets []string

// SecretValues is a map containing the actual values of the secrets. Note that this should not be written
// to kubernetes or anywhere else persistent!
type SecretValues map[string][]byte

// SecretsProvider reads secrets for a release from both the desired state as well as the current state
type SecretsProvider interface {
	Read(componentName, namespace string) (SecretValues, error)
	Write(componentName, namespace string, secretValues SecretValues) error
	Delete(componentName, namespace string) error
}

type secretsProvider struct {
	env *Environment
}

// NewSecretsProvider is a factory method to create a new SecretsProvider
func NewSecretsProvider(env *Environment) SecretsProvider {
	return &secretsProvider{env: env}
}

func (sp *secretsProvider) Read(componentName, namespace string) (SecretValues, error) {
	logrus.WithFields(logrus.Fields{"component": componentName, "namespace": namespace}).Debug("Reading secrets for component")

	secrets := SecretValues{}

	secret, err := sp.env.KubeClient().Secrets(namespace).Get(componentName)
	if err != nil {
		if errors.IsNotFound(err) {
			logrus.WithFields(logrus.Fields{"component": componentName, "namespace": namespace}).Debug("No secrets found for component")
			return secrets, nil
		}

		logrus.WithFields(logrus.Fields{
			"component": componentName,
			"namespace": namespace,
			"error":     err,
		}).Error("Error when reading secrets for component")
		return nil, err
	}

	for key, val := range secret.Data {
		secrets[key] = val
	}

	logrus.WithFields(logrus.Fields{"component": componentName, "namespace": namespace}).Debug("Successfully read secrets for component")

	return secrets, nil
}

func (sp *secretsProvider) Write(componentName, namespace string, secrets SecretValues) error {
	logrus.WithFields(logrus.Fields{"component": componentName, "namespace": namespace}).Info("Writing secrets for component")

	err := sp.ensureNamespace(namespace)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"component": componentName,
			"namespace": namespace,
			"error":     err,
		}).Error("Error when ensuring namespace exists for secret")
		return err
	}

	_, err = sp.env.KubeClient().Secrets(namespace).Create(&api.Secret{
		ObjectMeta: api.ObjectMeta{
			Name: componentName,
		},
		Data: secrets,
	})
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"component": componentName,
			"namespace": namespace,
			"error":     err,
		}).Error("Error when writing secrets for component")
		return err
	}

	logrus.WithFields(logrus.Fields{"component": componentName, "namespace": namespace}).Info("Successfully written secrets for component")

	return nil
}

func (sp *secretsProvider) Delete(componentName, namespace string) error {
	logrus.WithFields(logrus.Fields{"component": componentName, "namespace": namespace}).Info("Deleting existing secrets for component")

	// We first completely delete the current secrets
	err := sp.env.KubeClient().Secrets(namespace).Delete(componentName, nil)
	if err != nil {
		if errors.IsNotFound(err) {
			logrus.WithFields(logrus.Fields{"component": componentName, "namespace": namespace}).Info("No secrets found for component")
			return nil
		}

		logrus.WithFields(logrus.Fields{
			"component": componentName,
			"namespace": namespace,
			"error":     err,
		}).Error("Error when deleting current secrets for component")
		return err
	}

	return nil
}

// ensureNamespace trigger namespace creation and filter errors, only already-exists type of error won't be returned.
func (sp *secretsProvider) ensureNamespace(namespace string) error {
	_, err := sp.env.KubeClient().Namespaces().Create(
		&api.Namespace{
			ObjectMeta: api.ObjectMeta{
				Name: namespace,
			},
		},
	)

	if errors.IsAlreadyExists(err) {
		return nil
	}

	return err
}

// readSecretValues obtains secrets from environment variables
func readSecretValues(cmp *Component) {
	for _, key := range cmp.Secrets {
		envName := strings.Replace(strings.ToUpper(key), "-", "_", -1)

		secretValue := os.Getenv(envName)
		if len(secretValue) == 0 {
			logrus.WithFields(logrus.Fields{"secret": key, "envName": envName}).Warn("Secret not found in environment")
		}

		cmp.SecretValues[key] = []byte(secretValue)
	}
}
