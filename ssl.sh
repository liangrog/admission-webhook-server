#!/bin/bash
: ${1?'missing CN'}
cn="$1"

secret_dir="helm/ssl"
expiration="3650"
mkdir -p helm/ssl

chmod 0700 "$secret_dir"
cd "$secret_dir"

rm -rf *

# Generate the CA cert and private key
openssl req -nodes -new -x509 -days $expiration -keyout ca.key -out ca.crt -subj "/CN=Admission Controller Webhook Server CA"

cat ca.key > server.pem
cat ca.crt >> server.pem

# Generate the private key for the webhook server
openssl genrsa -out tls.key 2048
# Generate a Certificate Signing Request (CSR) for the private key, and sign it with the private key of the CA.
# https://security.stackexchange.com/a/190906
openssl req -new -days $expiration -key tls.key -subj "/CN=$cn" \
    | openssl x509 -days $expiration -req -CA ca.crt -CAkey ca.key -CAcreateserial -out tls.crt \
        -extensions SAN -extfile <(cat /etc/ssl/openssl.cnf <(printf "\n[SAN]\nsubjectAltName=DNS:$cn"))
