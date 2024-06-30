# Kubernetes Webhook for Certificate Management

This project provides a Kubernetes webhook service designed to manage and automate the handling of certificates within a Kubernetes cluster. It leverages custom mutation and validation webhooks to ensure that certificates are correctly issued, renewed, and injected into the appropriate Kubernetes resources.

## Features

- **Certificate Caching**: Implements a caching mechanism to optimize certificate retrieval and reduce the load on certificate issuers.
- **Azure Key Vault Integration**: Utilizes Azure Key Vault for secure storage and management of certificates.
- **Certificate Manager Wrapper**: Provides a simplified interface for interacting with cert-manager, making it easier to request and manage certificates.
- **Kubernetes Client**: Includes a Kubernetes client for interacting with the Kubernetes API, facilitating the creation and management of resources such as MutatingWebhookConfiguration and ValidatingWebhookConfiguration.
- **Utility Scripts**: Contains utility scripts like `gen-certs.sh` for generating certificates required for the webhook service to operate securely.

## Getting Started

### Prerequisites

- Kubernetes cluster
- Docker
- Go 1.15 or higher