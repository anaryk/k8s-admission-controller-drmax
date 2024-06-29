package azurewrapper

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"time"

	"dev.azure.com/drmaxglobal/devops-team/_git/k8s-system-operator/pkg/k8s"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/keyvault/azsecrets"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type KeyVaultClient struct {
	client *azsecrets.Client
}

func NewKeyVaultClient(vaultName string) (*KeyVaultClient, error) {
	vaultURL := fmt.Sprintf("https://%s.vault.azure.net/", vaultName)
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to obtain a credential: %w", err)
	}

	client, err := azsecrets.NewClient(vaultURL, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create secret client: %w", err)
	}

	return &KeyVaultClient{client: client}, nil
}

func (kvc *KeyVaultClient) StoreSecret(ctx context.Context, secretName string, cert, key []byte) error {
	secretValue := fmt.Sprintf("%s\n%s", string(cert), string(key))
	_, err := kvc.client.SetSecret(ctx, secretName, azsecrets.SetSecretParameters{Value: &secretValue}, nil)
	if err != nil {
		return fmt.Errorf("failed to store secret: %w", err)
	}
	return nil
}

func (kvc *KeyVaultClient) GetSecret(ctx context.Context, secretName string) ([]byte, []byte, error) {
	resp, err := kvc.client.GetSecret(ctx, secretName, "", nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get secret: %w", err)
	}

	secretValue := resp.Value
	cert, key := parseCertAndKey([]byte(*secretValue))
	return cert, key, nil
}

func (kvc *KeyVaultClient) GetCertificateExpiry(ctx context.Context, secretName string) (time.Time, error) {
	cert, _, err := kvc.GetSecret(ctx, secretName)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get secret: %w", err)
	}

	block, _ := pem.Decode(cert)
	if block == nil || block.Type != "CERTIFICATE" {
		return time.Time{}, fmt.Errorf("failed to parse certificate PEM")
	}

	parsedCert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse certificate: %w", err)
	}

	return parsedCert.NotAfter, nil
}

func (kvc *KeyVaultClient) DeleteSecret(ctx context.Context, secretName string) error {
	_, err := kvc.client.DeleteSecret(ctx, secretName, nil)
	if err != nil {
		return fmt.Errorf("failed to delete secret: %w", err)
	}
	return nil
}

func (kvc *KeyVaultClient) SecretExists(ctx context.Context, secretName string) (bool, error) {
	_, err := kvc.client.GetSecret(ctx, secretName, "", nil)
	if err != nil {
		return false, fmt.Errorf("secret not exist in target keyvault: %w", err)
	}
	return true, nil
}

func (kvc *KeyVaultClient) SaveSecretToK8s(ctx context.Context, secretName, secretNameKube, namespace string) error {
	cert, key, err := kvc.GetSecret(ctx, secretName)
	if err != nil {
		return fmt.Errorf("failed to get secret from key vault: %w", err)
	}
	client, err := k8s.PrepareInClusterK8SClient()
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(client)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes clientset: %w", err)
	}

	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretNameKube,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"tls.crt": cert,
			"tls.key": key,
		},
		Type: v1.SecretTypeTLS,
	}

	_, err = clientset.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes secret: %w", err)
	}
	return nil
}

func parseCertAndKey(secretValue []byte) ([]byte, []byte) {
	parts := bytes.Split(secretValue, []byte("\n"))
	if len(parts) < 2 {
		return nil, nil
	}
	return parts[0], parts[1]
}
