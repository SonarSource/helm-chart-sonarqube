create_temp_dir() {
  TEMP_DIR=$(mktemp -d)
  echo "Temporary directory created at: $TEMP_DIR"

  # Ensure the temporary directory is removed when the script exits
  trap 'echo "Removing temporary directory: $TEMP_DIR"; rm -rf "$TEMP_DIR"' EXIT
}

gen_kubeconfig() {
  echo "Generating kubeconfig..."
  acp_host=$1
  acp_token=$2
  cluster=$3
  kubeconfig_file=$4

  cat >"$kubeconfig_file" <<EOF
apiVersion: v1
clusters:
- cluster:
    insecure-skip-tls-verify: true
    server: $acp_host/kubernetes/$cluster
  name: $cluster-cluster
contexts:
- context:
    cluster: $cluster-cluster
    user: user-$cluster-cluster
  name: $cluster-cluster
current-context: $cluster-cluster
kind: Config
users:
- name: user-$cluster-cluster
  user:
    token: $acp_token
EOF
}

gen_kubeconfig_base_config() {
  echo "Generating kubeconfig base config yaml..."
  config_yaml=$1
  kubeconfig_file_path=$2

  # Using backticks instead of $() for POSIX compatibility
  acp_host=$(yq '.acp.baseUrl' "$config_yaml")
  acp_token=$(yq '.acp.token' "$config_yaml")
  acp_test_cluster=$(yq '.acp.cluster' "$config_yaml")

  echo "ACP Host: $acp_host"
  echo "ACP Token: *****"
  echo "ACP Test Cluster: $acp_test_cluster"

  gen_kubeconfig "$acp_host" "$acp_token" "$acp_test_cluster" "$kubeconfig_file_path"
  echo "Kubeconfig base config yaml generated at: $kubeconfig_file_path"
}