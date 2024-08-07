package azurewrapper

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"
	"time"

	"dev.azure.com/drmaxglobal/devops-team/_git/k8s-system-operator/pkg/k8s"
	"dev.azure.com/drmaxglobal/devops-team/_git/k8s-system-operator/pkg/utils"
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

	expiry, err := utils.GetFirstCertExpiryFromPEM(cert)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse certificate: %w", err)
	}

	return expiry, nil
}

func (kvc *KeyVaultClient) GetCertificateDetails(ctx context.Context, secretName string) (string, []string, error) {
	cert, _, err := kvc.GetSecret(ctx, secretName)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get secret: %w", err)
	}

	commonName, altNames, err := getFirstCertDetailsFromPEM(cert)
	if err != nil {
		return "", nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	return commonName, altNames, nil
}

func getFirstCertDetailsFromPEM(certPEM []byte) (string, []string, error) {
	block, _ := pem.Decode(certPEM)
	if block == nil || block.Type != "CERTIFICATE" {
		return "", nil, fmt.Errorf("failed to parse certificate PEM")
	}

	parsedCert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	return parsedCert.Subject.CommonName, parsedCert.DNSNames, nil
}

func (kvc *KeyVaultClient) DeleteSecret(ctx context.Context, secretName string) error {
	//Validate if exist
	if exists, _ := kvc.SecretExists(ctx, secretName); !exists {
		return nil
	}

	//Delete secret
	_, err := kvc.client.DeleteSecret(ctx, secretName, nil)
	if err != nil {
		return fmt.Errorf("failed to delete secret: %w", err)
	}
	return nil
}

func (kvc *KeyVaultClient) ListSecretsPendingPurge(ctx context.Context) ([]string, error) {
	pager := kvc.client.NewListDeletedSecretsPager(nil)
	var secretsPendingPurge []string

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list deleted secrets: %w", err)
		}

		for _, secret := range page.Value {
			secretID := string(*secret.ID)
			secretIDParts := strings.Split(secretID, "/")
			secretName := secretIDParts[len(secretIDParts)-1]
			secretsPendingPurge = append(secretsPendingPurge, secretName)
		}
	}

	return secretsPendingPurge, nil
}

func (kvc *KeyVaultClient) PurgerDeletedSecret(ctx context.Context, secretName string) error {
	_, err := kvc.client.PurgeDeletedSecret(ctx, secretName, nil)
	if err != nil {
		return fmt.Errorf("failed to purge secret: %w", err)
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

	commonName, altNames, err := getFirstCertDetailsFromPEM(cert)
	if err != nil {
		return fmt.Errorf("failed to parse certificate: %w", err)
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
			Labels: map[string]string{
				"controller.cert-manager.io/fao": "true",
			},
			Annotations: map[string]string{
				"cert-manager.io/alt-names":        strings.Join(altNames, ","),
				"cert-manager.io/common-name":      commonName,
				"cert-manager.io/certificate-name": secretNameKube,
				"cert-manager.io/ip-sans":          "",
				"cert-manager.io/uri-sans":         "",
				"cert-manager.io/issuer-name":      "cert-manager",
				"cert-manager.io/issuer-kind":      "ClusterIssuer",
				"cert-manager.io/issuer-group":     "cert-manager.io",
			},
		},
		Data: map[string][]byte{
			"tls.crt": cert,
			"tls.key": key,
		},
		Type: v1.SecretTypeTLS,
	}

	existingSecret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, secretNameKube, metav1.GetOptions{})
	if err != nil {
		_, err = clientset.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("failed to create Kubernetes secret: %w", err)
		}
	} else {
		existingSecret.Data = secret.Data
		existingSecret.Labels = secret.Labels
		existingSecret.Annotations = secret.Annotations
		_, err = clientset.CoreV1().Secrets(namespace).Update(ctx, existingSecret, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to update Kubernetes secret: %w", err)
		}
	}

	return nil
}

func parseCertAndKey(secretValue []byte) ([]byte, []byte) {
	parts := bytes.Split(secretValue, []byte("\n"))
	var certBuffer bytes.Buffer
	var keyBuffer bytes.Buffer
	inCert := false
	inKey := false

	for _, part := range parts {
		line := string(part)
		if strings.HasPrefix(line, "-----BEGIN CERTIFICATE-----") {
			inCert = true
		}
		if strings.HasPrefix(line, "-----BEGIN PRIVATE KEY-----") || strings.HasPrefix(line, "-----BEGIN RSA PRIVATE KEY-----") {
			inKey = true
			inCert = false
		}
		if inCert {
			certBuffer.WriteString(line + "\n")
		} else if inKey {
			keyBuffer.WriteString(line + "\n")
		}
		if strings.HasPrefix(line, "-----END CERTIFICATE-----") {
			certBuffer.WriteString(line + "\n")
			inCert = false
		}
		if strings.HasPrefix(line, "-----END PRIVATE KEY-----") || strings.HasPrefix(line, "-----END RSA PRIVATE KEY-----") {
			keyBuffer.WriteString(line + "\n")
			inKey = false
		}
	}

	// Remove duplicate ending tags
	certPEM := certBuffer.String()
	certPEM = strings.ReplaceAll(certPEM, "\n-----END CERTIFICATE-----\n-----END CERTIFICATE-----", "\n-----END CERTIFICATE-----")
	keyPEM := keyBuffer.String()
	keyPEM = strings.ReplaceAll(keyPEM, "\n-----END PRIVATE KEY-----\n-----END PRIVATE KEY-----", "\n-----END PRIVATE KEY-----")
	keyPEM = strings.ReplaceAll(keyPEM, "\n-----END RSA PRIVATE KEY-----\n-----END RSA PRIVATE KEY-----", "\n-----END RSA PRIVATE KEY-----")

	return []byte(certPEM), []byte(keyPEM)
}
