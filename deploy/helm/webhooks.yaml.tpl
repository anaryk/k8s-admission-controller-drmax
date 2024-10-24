apiVersion: admissionregistration.k8s.io/v1
kind: MutatingWebhookConfiguration
metadata:
  name: k8s-admission-webhook-drmax
webhooks:
  - name: certorder.drmax.global
    admissionReviewVersions: ["v1"]
    sideEffects: None
    clientConfig:
      service:
        name: k8s-admission-webhook-drmax
        namespace: k8s-admission-controller-drmax
        path: /webhooks/mutating/certorder
      caBundle: CA_BUNDLE
    rules:
      - operations: ["CREATE", "UPDATE"]
        apiGroups: ["acme.cert-manager.io"]
        apiVersions: ["v1"]
        resources: ["challenges", "challenges/status"]
  - name: ingresscerts.drmax.global
    admissionReviewVersions: ["v1"]
    sideEffects: None
    clientConfig:
      service:
        name: k8s-admission-webhook-drmax
        namespace: k8s-admission-controller-drmax
        path: /webhooks/mutating/ingresscerts
      caBundle: CA_BUNDLE
    rules:
      - operations: ["CREATE", "UPDATE"]
        apiGroups: ["networking.k8s.io"]
        apiVersions: ["v1"]
        resources: ["ingresses"]
  - name: certificatecache.drmax.global
    admissionReviewVersions: ["v1"]
    sideEffects: None
    clientConfig:
      service:
        name: k8s-admission-webhook-drmax
        namespace: k8s-admission-controller-drmax
        path: /webhooks/mutating/certificatecache
      caBundle: CA_BUNDLE
    rules:
      - operations: ["CREATE"]
        apiGroups: ["cert-manager.io"]
        apiVersions: ["v1"]
        resources: ["certificates"]