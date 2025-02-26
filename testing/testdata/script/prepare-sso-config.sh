#!/bin/bash
set -e

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
source ${SCRIPT_DIR}/common.sh

# Main function to prepare SSO configuration
prepare_sso_config() {
  create_temp_dir

  acp_host=$1
  acp_token=$2
  cluster=$3
  test_ns=${4:-"testing-sonarqube-operator"}
  sonarqube_base_url=${5:-"https://test-https-sso.example.com"}

  echo "acp_host: $acp_host"
  echo "acp_token: ******"
  echo "cluster: $cluster"

  if [ -z "$acp_host" ] || [ -z "$acp_token" ] || [ -z "$cluster" ]; then
    echo "请指定 acp_host, acp_token 和 cluster 参数"
    exit 1
  fi

  echo "generate kubeconfig yamls ..."
  business_kubeconfig="${TEMP_DIR}/business_kubeconfig.yaml"
  global_kubeconfig="${TEMP_DIR}/global_kubeconfig.yaml"
  gen_kubeconfig "${acp_host}" "${acp_token}" "${cluster}" "${business_kubeconfig}"
  gen_kubeconfig "${acp_host}" "${acp_token}" "global" "${global_kubeconfig}"

  # 创建 Oauth2Client 资源
  echo "creating Oauth2Client resource ..."
  KUBECONFIG=${global_kubeconfig} kubectl apply -f - <<EOF
apiVersion: dex.coreos.com/v1
kind: OAuth2Client
name: OIDC
metadata:
  name: orsxg5bnmrsxrs7sttsiiirdeu
  namespace: cpaas-system
id: test-dex
public: false
redirectURIs:
  - ${sonarqube_base_url}/*
secret: Z2l0bGFiLW9mZmljaWFsLTAK
spec: {}
EOF

  # 获取 dex 证书
  echo "creating dex certificate ..."
  dex_tls_yaml="${TEMP_DIR}/dex-tls.yaml"
  KUBECONFIG=${global_kubeconfig} kubectl get secret -n cpaas-system dex.tls -o yaml >"${dex_tls_yaml}"
  KUBECONFIG=${business_kubeconfig} kubectl apply -f - <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: dex-tls
  namespace: ${test_ns}
data:
  ca.crt: $(yq '.data."ca.crt"' <"${dex_tls_yaml}")
  tls.crt: $(yq '.data."tls.crt"' <"${dex_tls_yaml}")
EOF

  echo "SSO configuration preparation completed."
}

# Execute the main function with all script arguments
prepare_sso_config "$@"