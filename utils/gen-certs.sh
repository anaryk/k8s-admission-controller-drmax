#! /bin/bash

WEBHOOK_NS=k8s-admission-controller-drmax
WEBHOOK_NAME=k8s-admission-webhook-drmax
WEBHOOK_SVC=${WEBHOOK_NAME}.${WEBHOOK_NS}.svc
K8S_OUT_CERT_FILE=./deploy/app-certs.yaml
K8S_OUT_WEBBOK_FILE=./deploy/webhooks.yaml

OUT_CERT="./webhook.crt"
OUT_KEY="./webhookCA.key"
   
# Create certs for our webhook 
set -f 
mkcert \
  --cert-file "${OUT_CERT}" \
  --key-file "${OUT_KEY}" \
  "${WEBHOOK_SVC}"
set +f

# Create certs secrets for k8s.
rm ${K8S_OUT_CERT_FILE}
kubectl -n ${WEBHOOK_NS} create secret generic \
    ${WEBHOOK_NAME}-certs \
    --from-file=key.pem=${OUT_KEY} \
    --from-file=cert.pem=${OUT_CERT}\
    --dry-run=client -o yaml > ${K8S_OUT_CERT_FILE}

# Set the CABundle on the webhook registration.
CA_BUNDLE=$(cat ./${OUT_CERT} | base64 -b 0)
sed "s/CA_BUNDLE/${CA_BUNDLE}/" ./deploy/webhooks.yaml.tpl > ${K8S_OUT_WEBBOK_FILE}

# Clean.
rm "${OUT_CERT}" && rm "${OUT_KEY}"