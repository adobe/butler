#!/bin/bash
# Script to generate test CA and certificates with proper SANs for butler acceptance tests

set -e

CERT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$CERT_DIR"

echo "Generating certificates in: $CERT_DIR"

# Clean up old files
rm -f rootCA.crt rootCA.key rootCA.srl test.crt test.key test.csr rootCA.ky test.ky

# Generate Root CA private key (4096 bit RSA)
echo "Generating Root CA private key..."
openssl genrsa -out rootCA.key 4096

# Generate Root CA certificate (self-signed, valid for ~56 years like the original)
echo "Generating Root CA certificate..."
openssl req -x509 -new -nodes \
    -key rootCA.key \
    -sha256 \
    -days 20454 \
    -out rootCA.crt \
    -subj "/C=UK/ST=Berkshire/L=Maidenhead/O=Adobe Systems, Ltd/OU=TechOps"

# Create OpenSSL config for server certificate with SANs
cat > server.cnf << 'EOF'
[req]
default_bits = 4096
prompt = no
default_md = sha256
distinguished_name = dn
req_extensions = req_ext

[dn]
C = UK
ST = Berkshire
L = Maidenhead
O = Adobe Systems, Ltd
OU = TechOps
CN = localhost

[req_ext]
subjectAltName = @alt_names

[alt_names]
DNS.1 = localhost
DNS.2 = *.localhost
IP.1 = 127.0.0.1
IP.2 = ::1
EOF

# Generate server private key
echo "Generating server private key..."
openssl genrsa -out test.key 4096

# Generate server CSR with SANs
echo "Generating server CSR..."
openssl req -new \
    -key test.key \
    -out test.csr \
    -config server.cnf

# Create extension file for signing (needed to include SANs in the signed cert)
cat > server_ext.cnf << 'EOF'
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, nonRepudiation, keyEncipherment, dataEncipherment
subjectAltName = @alt_names

[alt_names]
DNS.1 = localhost
DNS.2 = *.localhost
IP.1 = 127.0.0.1
IP.2 = ::1
EOF

# Sign the server certificate with the CA (valid for ~56 years like the original)
echo "Signing server certificate with CA..."
openssl x509 -req \
    -in test.csr \
    -CA rootCA.crt \
    -CAkey rootCA.key \
    -CAcreateserial \
    -out test.crt \
    -days 20454 \
    -sha256 \
    -extfile server_ext.cnf

# Create .ky copies for compatibility with existing code that expects .ky extension
cp rootCA.key rootCA.ky
cp test.key test.ky

# Clean up temporary config files
rm -f server.cnf server_ext.cnf

# Verify the certificates
echo ""
echo "=== Root CA Certificate ==="
openssl x509 -in rootCA.crt -text -noout | grep -A2 "Subject:"
echo ""
echo "=== Server Certificate ==="
openssl x509 -in test.crt -text -noout | grep -A2 "Subject:"
echo ""
echo "=== Server Certificate SANs ==="
openssl x509 -in test.crt -text -noout | grep -A1 "Subject Alternative Name"
echo ""
echo "=== Verification ==="
openssl verify -CAfile rootCA.crt test.crt

echo ""
echo "Certificate generation complete!"
echo "Files created:"
ls -la *.crt *.key *.ky *.csr *.srl 2>/dev/null || true
