# k8s-admission-controller-drmax Documentation

This documentation provides an in-depth overview of the `k8s-admission-controller-drmax` application. The application integrates with Kubernetes as an admission controller, handling both mutating and validating webhook configurations, managing certificates, and integrating securely with Azure KeyVault.

## Table of Contents

- [Project Overview](overview.md)
- [Webhooks Documentation](webhooks.md)
- [Certificate Cache Manager](certificate_cache_manager.md)
- [Azure KeyVault Integration](azure_keyvault.md)
- [Kubernetes Client Interactions](kubernetes_client.md)
- [Utility Functions](utility_functions.md)

## Getting Started

### Prerequisites

- Go version 1.16 or higher
- Docker for containerization
- Access to a Kubernetes cluster
- Azure KeyVault for secret management

### Installation

Clone the repository and navigate to the project directory:

```bash
git clone https://github.com/your-repo/k8s-admission-controller-drmax.git
cd k8s-admission-controller-drmax
```

Build the Docker image:

```bash
docker build -t k8s-admission-controller-drmax:latest .
```

Deploy the controller to your Kubernetes cluster:

```bash
kubectl apply -f deployment.yaml
```

## Contribution Guidelines

Contributions to this project are welcome. Please fork the repository and create a pull request with your changes.

## License

This project is licensed under the MIT License.
