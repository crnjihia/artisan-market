#!/usr/bin/env bash
# path: scripts/generate-dev-certs.sh
set -euo pipefail

CERTS_DIR="${1:-certs}"
mkdir -p "${CERTS_DIR}"

echo "==> Generating Development Certificate Authority (Root CA)..."
openssl genrsa -out "${CERTS_DIR}/ca.key" 4096 2>/dev/null
openssl req -x509 -new -nodes -key "${CERTS_DIR}/ca.key" -sha256 -days 365 \
  -subj "/C=KE/ST=Nairobi/L=Nairobi/O=Artisan Market/OU=Security/CN=ArtisanMarketRootCA" \
  -out "${CERTS_DIR}/ca.crt" 2>/dev/null

SERVICES=("gateway" "user" "product" "order" "payment")

for SVC in "${SERVICES[@]}"; do
  echo "==> Generating mTLS certificate for ${SVC}..."
  openssl genrsa -out "${CERTS_DIR}/${SVC}.key" 2048 2>/dev/null

  # Generate Certificate Signing Request
  openssl req -new -key "${CERTS_DIR}/${SVC}.key" \
    -subj "/C=KE/ST=Nairobi/L=Nairobi/O=Artisan Market/OU=${SVC}/CN=${SVC}" \
    -out "${CERTS_DIR}/${SVC}.csr" 2>/dev/null

  # Sign with Root CA with Subject Alternative Names (SAN)
  cat > "${CERTS_DIR}/${SVC}.ext" <<EOF
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, nonRepudiation, keyEncipherment, dataEncipherment
extendedKeyUsage = serverAuth, clientAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = ${SVC}
DNS.2 = ${SVC}.artisan-market.svc.cluster.local
DNS.3 = localhost
IP.1 = 127.0.0.1
EOF

  openssl x509 -req -in "${CERTS_DIR}/${SVC}.csr" \
    -CA "${CERTS_DIR}/ca.crt" -CAkey "${CERTS_DIR}/ca.key" -CAcreateserial \
    -out "${CERTS_DIR}/${SVC}.crt" -days 365 -sha256 \
    -extfile "${CERTS_DIR}/${SVC}.ext" 2>/dev/null

  rm -f "${CERTS_DIR}/${SVC}.csr" "${CERTS_DIR}/${SVC}.ext"
done

echo "==> Dev certificates generated successfully in ${CERTS_DIR}/"
ls -la "${CERTS_DIR}"

