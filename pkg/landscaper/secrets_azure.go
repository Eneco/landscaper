package landscaper

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"

	"github.com/Azure/azure-sdk-for-go/arm/examples/helpers"
	"github.com/Azure/azure-sdk-for-go/dataplane/keyvault"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
)

type AzureSecretsReader struct{
	kvClient keyvault.ManagementClient
	kvName string
	kvURL string
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

	kvDNSSuffix := os.Getenv("AZURE_KEYVAULT_DNS_SUFFIX")
	if kvDNSSuffix == "" {
		kvDNSSuffix = azure.PublicCloud.KeyVaultDNSSuffix
	}
	
	resource := fmt.Sprintf("https://%s", kvDNSSuffix)
	spt, err := helpers.NewServicePrincipalTokenFromCredentials(envVars, resource)
	if err != nil {
		logrus.WithField("error", err).Fatalf("Failed to get service principle token")
		return nil, fmt.Errorf("failed to get azure service principal token")
	}

	kv := keyvault.New()
	kv.Authorizer = autorest.NewBearerAuthorizer(spt)

	baseURL := fmt.Sprintf("https://%s.%s/", keyVault, kvDNSSuffix)

	return &AzureSecretsReader{kvClient: kv, kvName: keyVault, kvURL: baseURL}, nil
}

// Reads the secret from the Azure key vault
func (asp *AzureSecretsReader) Read(componentName, namespace string, secretNames []string) (SecretValues, error) {
	logrus.WithFields(logrus.Fields{"component": componentName, "namespace": namespace}).Debug("Reading secrets for component")
	
	secrets := SecretValues{}

	for _, secret := range secretNames {
		value, err := asp.kvClient.GetSecret(asp.kvURL, secret, "")
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"component": componentName, 
				"namespace": namespace,
				"keyvault": asp.kvURL, 
				"secret": secret, 
				"error": err,
			}).Error("Failed to get secret from keyvault",)
			return secrets, fmt.Errorf("failed to get secret `%s` from keyvault `%s`", secret, asp.kvURL)
		}
		secrets[secret] = []byte(*value.Value)
	}

	logrus.WithFields(logrus.Fields{
		"component": componentName, 
		"namespace": namespace,
	}).Debug("Successfully read secrets for component")

	return secrets, nil
}