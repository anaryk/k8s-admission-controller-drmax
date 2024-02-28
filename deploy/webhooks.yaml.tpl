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
        namespace: k8s-admission-webhook-drmax
        path: /webhooks/mutating/certorder
      caBundle: CA_BUNDLE
    rules:
      - operations: ["CREATE", "UPDATE"]
        apiGroups: ["acme.cert-manager.io"]
        apiVersions: ["v1"]
        resources: ["challenges", "challenges/status"]
  - name: spotscaler.drmax.global
    admissionReviewVersions: ["v1"]
    sideEffects: None
    clientConfig:
      service:
        name: k8s-admission-webhook-drmax
        namespace: k8s-admission-webhook-drmax
        path: /webhooks/mutating/spotscaler
      caBundle: CA_BUNDLE
    rules:
      - operations: ["*"]
        apiGroups: ["apps"]
        apiVersions: ["v1"]
        resources: ["deployments/scale"]