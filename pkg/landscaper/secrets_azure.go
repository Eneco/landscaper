package landscaper

import (
	"fmt"
	"os"

	"github.com/Sirupsen/logrus"

	"github.com/Azure/azure-sdk-for-go/arm/examples/helpers"
	"github.com/Azure/azure-sdk-for-go/dataplane/keyvault"
	"github.com/Azure/go-autorest/autorest"
)

type AzureSecretsReader struct{
	kvClient keyvault.ManagementClient
	kvName string
}

func NewAzureSecretsReader(keyVault string) (SecretsReader, error) {
	envVars := map[string]string{
		"AZURE_CLIENT_ID":       os.Getenv("AZURE_CLIENT_ID"),
		"AZURE_CLIENT_SECRET":   os.Getenv("AZURE_CLIENT_SECRET"),
		"AZURE_TENANT_ID":       os.Getenv("AZURE_TENANT_ID"),
	}

	for varName, value := range envVars {
		if value == "" {
			logrus.WithField("variable", varName).Fatalf("Missing environment variable")
			return nil, fmt.Errorf("azure client environment variable `%s` is empty", varName)
		}
	}
	
	resource := "https://vault.azure.net"
	spt, err := helpers.NewServicePrincipalTokenFromCredentials(envVars, resource)
	if err != nil {
		logrus.WithField("error", err).Fatalf("Failed to get service principle token")
		return nil, fmt.Errorf("failed to get azure service principal token")
	}

	kv := keyvault.New()
	kv.Authorizer = autorest.NewBearerAuthorizer(spt)

	return &AzureSecretsReader{kvClient: kv, kvName: keyVault}, nil
}

// Reads the secret from the Azure key vault
func (asp *AzureSecretsReader) Read(componentName, namespace string, secretNames []string) (SecretValues, error) {
	logrus.WithFields(logrus.Fields{"component": componentName, "namespace": namespace}).Debug("Reading secrets for component")
	
	secrets := SecretValues{}

	baseURL := fmt.Sprintf("https://%s.vault.azure.net/", asp.kvName)

	for _, secret := range secretNames {
		value, err := asp.kvClient.GetSecret(baseURL, secret, "")
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"component": componentName, 
				"namespace": namespace,
				"keyvault": asp.kvName, 
				"secret": secret, 
			}).Error("Secret not found in keyvault")
			return secrets, fmt.Errorf("secret `%s` was not found in keyvault `%s`", secret, asp.kvName)
		}
		secrets[secret] = []byte(*value.Value)
	}

	logrus.WithFields(logrus.Fields{
		"component": componentName, 
		"namespace": namespace,
	}).Debug("Successfully read secrets for component")

	return secrets, nil
}