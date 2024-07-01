#!/bin/bash

# Array of namespaces
namespaces=(
    "ro-catalog"
    "ro-checkout"
    "ro-content"
    "ro-customer"
    "ro-mainframe"
    "ro-marketplace"
    "ro-platform"
    "ro-sre"
    "sk-catalog"
    "sk-checkout"
    "sk-content"
    "sk-customer"
    "sk-mainframe"
    "sk-marketplace"
    "sk-platform"
    "sk-sre"
    "pl-catalog"
    "pl-checkout"
    "pl-content"
    "pl-customer"
    "pl-mainframe"
    "pl-marketplace"
    "pl-platform"
    "pl-sre"
    "it-catalog"
    "it-checkout"
    "it-content"
    "it-customer"
    "it-marketplace"
    "it-platform"
    "it-sre"
    "cz-catalog"
    "cz-checkout"
    "cz-content"
    "cz-customer"
    "cz-devops"
    "cz-mainframe"
    "cz-marketplace"
    "cz-platform"
    "cz-sre"
)

# Annotation to be added
annotation="admissions.drmax.gl/cache-certs=true"

# Iterate through each namespace
for ns in "${namespaces[@]}"; do
    echo "Processing namespace: $ns"
    # Get all Ingress names in the current namespace
    ingress_list=$(kubectl get ingress -n $ns -o jsonpath='{.items[*].metadata.name}')
    
    # Iterate through each ingress in the namespace
    for ingress in $ingress_list; do
        echo "  Adding annotation to ingress: $ingress"
        # Add the annotation to the ingress
        kubectl annotate ingress $ingress -n $ns $annotation --overwrite
    done
done

echo "Annotations have been added to all ingress objects."

