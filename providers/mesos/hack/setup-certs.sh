#!/bin/bash

set -e
set -x

# HACK_DIR contains the path to the directory where the current script lives.
HACK_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null && pwd )"
# DEST_DIR contains the path to the directory where the TLS material will be placed.
DEST_DIR="${HACK_DIR}/certs"

# Create DEST_DIR if it does not exist.
mkdir -p "${DEST_DIR}"

# =======
# Root CA
# =======

# Generate the root CA.
cfssl gencert -initca "${HACK_DIR}/ca-csr.json" | cfssljson -bare "${DEST_DIR}/ca"

# ==============
# kube-apiserver
# ==============

# Generate the certificate and private key for kube-apiserver.
cfssl gencert \
  -ca="${DEST_DIR}/ca.pem" \
  -ca-key="${DEST_DIR}/ca-key.pem" \
  -config="${HACK_DIR}/ca-config.json" \
  -hostname=kube-apiserver \
  -profile=kubernetes \
  "${HACK_DIR}/kube-apiserver-csr.json" | cfssljson -bare "${DEST_DIR}/kube-apiserver"

# Concatenate the CA certificate and the kube-apiserver certificate.
cat "${DEST_DIR}/kube-apiserver.pem" "${DEST_DIR}/ca.pem" > "${DEST_DIR}/kube-apiserver-crt.pem"

# =======================
# kube-controller-manager
# =======================

# Generate the keypair used to sign service account tokens.
cfssl gencert \
  -ca="${DEST_DIR}/ca.pem" \
  -ca-key="${DEST_DIR}/ca-key.pem" \
  -config="${HACK_DIR}/ca-config.json" \
  -profile=kubernetes \
  "${HACK_DIR}/service-account-csr.json" | cfssljson -bare "${DEST_DIR}/service-account"

# ===============
# virtual-kubelet
# ===============

# Generate the certificate and private key for virtual-kubelet.
cfssl gencert \
  -ca="${DEST_DIR}/ca.pem" \
  -ca-key="${DEST_DIR}/ca-key.pem" \
  -config="${HACK_DIR}/ca-config.json" \
  -hostname=virtual-kubelet \
  -profile=kubernetes \
  "${HACK_DIR}/virtual-kubelet-csr.json" | cfssljson -bare "${DEST_DIR}/virtual-kubelet"

# Concatenate the CA certificate and the virtual-kubelet certificate.
cat "${DEST_DIR}/virtual-kubelet.pem" "${DEST_DIR}/ca.pem" > "${DEST_DIR}/virtual-kubelet-crt.pem"
