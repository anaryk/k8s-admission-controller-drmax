# Project Overview

The `k8s-admission-controller-drmax` is a Kubernetes admission controller designed to enhance the security and efficiency of certificate management in Kubernetes environments. The controller includes both mutating and validating webhooks, which are essential for enforcing security policies and ensuring that Kubernetes resources are correctly configured before they are created or updated.

The application is structured into several key components, each with its own responsibilities:

- **Webhooks**: Handle the mutation and validation of Kubernetes resources.
- **Certificate Cache Manager**: Manages the caching and retrieval of certificates to improve performance and reduce redundant requests.
- **Azure KeyVault Integration**: Provides secure storage and retrieval of secrets and certificates from Azure KeyVault.
- **Kubernetes Client**: Facilitates interaction with the Kubernetes API for resource management.
- **Utility Functions**: Helper functions that perform common tasks such as certificate parsing and validation.
