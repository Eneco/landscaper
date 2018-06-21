package landscaper

import (
	"os"
	"strings"

	"fmt"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/kubernetes/pkg/apis/core"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset/typed/core/internalversion"
)

// SecretNames is a map containing the name of the secrets and a reference to get their value from the
// secrets provider.
type SecretNames map[string]string

// SecretValues is a map containing the actual values of the secrets. Note that this should not be written
// to kubernetes or anywhere else persistent!
type SecretValues map[string][]byte

// SecretsReader allows reading secrets
type SecretsReader interface {
	Read(componentName, namespace string, secretNames SecretNames) (SecretValues, error)
}

// SecretsWriteDeleter allows writing and deleting secrets
type SecretsWriteDeleter interface {
	Write(componentName, namespace string, secretValues SecretValues) error
	Delete(componentName, namespace string) error
}

// SecretsReadWriteDeleter allows reading, writing and deleting secrets
type SecretsReadWriteDeleter interface {
	SecretsReader
	SecretsWriteDeleter
}

type kubeSecretsProvider struct {
	kubeClient internalversion.CoreInterface
}

type environmentSecrets struct{}

// NewKubeSecretsReadWriteDeleter create a new SecretsReadWriteDeleter for kubernetes secrets
func NewKubeSecretsReadWriteDeleter(kubeClient internalversion.CoreInterface) SecretsReadWriteDeleter {
	return &kubeSecretsProvider{kubeClient: kubeClient}
}

// NewEnvironmentSecretsReader creates a SecretsReader for secrets provided via environment variables
func NewEnvironmentSecretsReader() SecretsReader {
	return &environmentSecrets{}
}

// Read reads all secrets in the k8s secret object. It ignores secretNames.
func (sp *kubeSecretsProvider) Read(componentName, namespace string, secretNames SecretNames) (SecretValues, error) {
	logrus.WithFields(logrus.Fields{"component": componentName, "namespace": namespace}).Debug("Reading secrets for component")

	secrets := SecretValues{}

	secret, err := sp.kubeClient.Secrets(namespace).Get(componentName, metav1.GetOptions{})
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

func (sp *kubeSecretsProvider) Write(componentName, namespace string, secrets SecretValues) error {
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

	_, err = sp.kubeClient.Secrets(namespace).Create(&core.Secret{
		ObjectMeta: metav1.ObjectMeta{
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

func (sp *kubeSecretsProvider) Delete(componentName, namespace string) error {
	logrus.WithFields(logrus.Fields{"component": componentName, "namespace": namespace}).Info("Deleting existing secrets for component")

	// We first completely delete the current secrets
	err := sp.kubeClient.Secrets(namespace).Delete(componentName, nil)
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
func (sp *kubeSecretsProvider) ensureNamespace(namespace string) error {
	_, err := sp.kubeClient.Namespaces().Create(
		&core.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		},
	)

	if errors.IsAlreadyExists(err) {
		return nil
	}

	return err
}

// Read reads the given secretNames from the environment, by uppercasing the name and converting - into _. componentName and namespace are ignored
func (env *environmentSecrets) Read(componentName, namespace string, secretNames SecretNames) (SecretValues, error) {
	secs := SecretValues{}
	for key, value := range secretNames {
		envName := strings.Replace(strings.ToUpper(value), "-", "_", -1)

		secretValue := os.Getenv(envName)
		if len(secretValue) == 0 {
			logrus.WithFields(logrus.Fields{"secret": key, "envName": envName}).Error("Secret not found in environment")
			return nil, fmt.Errorf("environment variable `%s` for secret `%s` empty", envName, key)
		}

		secs[key] = []byte(secretValue)
	}
	return secs, nil
}
