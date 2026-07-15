#!/bin/bash

# Helper function to display usage information
show_help() {
  cat << EOF
Ingress-NGINX to Gateway API (Istio) Migration Script for SonarQube

DESCRIPTION:
    Reads your current SonarQube Helm values (live release or a local file),
    detects the existing ingress/ingress-nginx configuration, and generates:
      - a Gateway API "Gateway" manifest, reusing any LoadBalancer Service
        annotations already configured under ingress-nginx.controller.service.annotations
        (the --aws-*/--gcp-*/--metallb-pool flags only add to or override
        what's already there — nothing is invented on your behalf)
      - a complete replacement values.yaml with ingress/ingress-nginx removed
        and httproute added, everything else preserved

    This script never runs "helm upgrade" against your SonarQube release. It
    only ever prints the command for you to run yourself, once you're ready.

USAGE:
    $0 --cloud aws|gcp|onprem [OPTIONS]

OPTIONS:
    --cloud aws|gcp|onprem   Target environment, selects Gateway annotations (REQUIRED)
    --mode generate|apply    generate: render files only (default)
                             apply: also install Gateway API CRDs + Istio (unless
                             --skip-istio) and kubectl apply the Gateway manifest
    --namespace ns           Kubernetes namespace of the release (default: sonarqube)
    --release-name name      Helm release name (default: sonarqube)
    --chart-flavor flavor    sonarqube|sonarqube-dce, used to derive the backend
                             service name (default: sonarqube)
    --chart-ref ref          Chart reference used only in the printed next-step
                             "helm upgrade" command (default: sonarqube/<chart-flavor>)
    --values-file path       Read values from this local file instead of
                              "helm get values <release> -n <namespace>"
    --hostnames h1,h2        Hostnames for the Gateway/HTTPRoute. Auto-detected
                              from ingress.hosts if omitted
    --gateway-name name      Name for the generated Gateway (default: <release-name>)
    --gateway-namespace ns   Namespace for the generated Gateway (default: <namespace>)
    --output-dir dir         Directory to write generated files into (default:
                              gateway-api-migration-<release-name>)
    --aws-subnets ids        Comma-separated subnet IDs for the AWS NLB. REQUIRED
                              for --cloud aws, unless already set via
                              ingress-nginx.controller.service.annotations
    --aws-scheme scheme      internet-facing|internal. Overrides the scheme if
                              one is already set on the existing Service
    --aws-cert-arn arn       ACM certificate ARN, enables the HTTPS listener
    --gcp-internal           Use an internal GCP load balancer
    --gcp-static-ip ip       Reserved static IP for the GCP load balancer
    --metallb-pool pool      MetalLB address pool name (on-prem only)
    --skip-istio             Assume Gateway API CRDs and Istio are already
                              installed (apply mode only)
    -h, --help               Show this help message and exit

EXAMPLES:
    # Show help
    $0 --help

    # Generate manifests for an EKS release, auto-detecting hostnames from the
    # live release's ingress config
    $0 --cloud aws --release-name sonarqube --namespace sonarqube \\
       --aws-subnets subnet-abc,subnet-def

    # Generate manifests offline, from a local values file
    $0 --cloud gcp --values-file ./my-values.yaml --hostnames sonarqube.example.com

    # Install CRDs/Istio and apply the Gateway for an on-prem cluster using MetalLB
    $0 --cloud onprem --mode apply --metallb-pool my-pool --release-name sonarqube

REQUIREMENTS:
    - kubectl configured and connected to the cluster (not required in generate
      mode when --values-file is used)
    - helm installed and configured (not required when --values-file is used)
    - yq (mikefarah/yq, v4+) available on PATH

EOF

  return 0
}

# Defaults
CLOUD=""
MODE="generate"
NAMESPACE="sonarqube"
RELEASE_NAME="sonarqube"
CHART_FLAVOR="sonarqube"
CHART_REF=""
VALUES_FILE=""
HOSTNAMES=""
GATEWAY_NAME=""
GATEWAY_NAMESPACE=""
OUTPUT_DIR=""
AWS_SUBNETS=""
AWS_SCHEME=""
AWS_CERT_ARN=""
GCP_INTERNAL="false"
GCP_STATIC_IP=""
METALLB_POOL=""
SKIP_ISTIO="false"

# Derived, computed once flags are parsed/validated
BACKEND_SERVICE=""
BACKEND_PORT=9000
CURRENT_VALUES_FILE=""
GATEWAY_FILE=""
NEW_VALUES_FILE=""
NEXT_STEP_CMD=""

# Filled in by introspect_ingress
TLS_ENABLED="false"
ANNOTATIONS_YAML="{}"

parse_args() {
  while [[ $# -gt 0 ]]; do
    case $1 in
      -h|--help)
        show_help
        exit 0
        ;;
      --cloud)
        CLOUD="$2"
        shift 2
        ;;
      --mode)
        MODE="$2"
        shift 2
        ;;
      --namespace)
        NAMESPACE="$2"
        shift 2
        ;;
      --release-name)
        RELEASE_NAME="$2"
        shift 2
        ;;
      --chart-flavor)
        CHART_FLAVOR="$2"
        shift 2
        ;;
      --chart-ref)
        CHART_REF="$2"
        shift 2
        ;;
      --values-file)
        VALUES_FILE="$2"
        shift 2
        ;;
      --hostnames)
        HOSTNAMES="$2"
        shift 2
        ;;
      --gateway-name)
        GATEWAY_NAME="$2"
        shift 2
        ;;
      --gateway-namespace)
        GATEWAY_NAMESPACE="$2"
        shift 2
        ;;
      --output-dir)
        OUTPUT_DIR="$2"
        shift 2
        ;;
      --aws-subnets)
        AWS_SUBNETS="$2"
        shift 2
        ;;
      --aws-scheme)
        AWS_SCHEME="$2"
        shift 2
        ;;
      --aws-cert-arn)
        AWS_CERT_ARN="$2"
        shift 2
        ;;
      --gcp-internal)
        GCP_INTERNAL="true"
        shift
        ;;
      --gcp-static-ip)
        GCP_STATIC_IP="$2"
        shift 2
        ;;
      --metallb-pool)
        METALLB_POOL="$2"
        shift 2
        ;;
      --skip-istio)
        SKIP_ISTIO="true"
        shift
        ;;
      *)
        echo "Unknown option: $1" >&2
        echo "Use -h or --help for usage information" >&2
        exit 1
        ;;
    esac
  done
}

validate_args() {
  if [[ -z "$CLOUD" ]]; then
    echo "Error: --cloud is required (aws|gcp|onprem)" >&2
    echo "Use -h or --help for detailed usage information" >&2
    exit 1
  fi

  if [[ "$CLOUD" != "aws" ]] && [[ "$CLOUD" != "gcp" ]] && [[ "$CLOUD" != "onprem" ]]; then
    echo "Error: --cloud must be one of aws|gcp|onprem, got '$CLOUD'" >&2
    exit 1
  fi

  if [[ "$MODE" != "generate" ]] && [[ "$MODE" != "apply" ]]; then
    echo "Error: --mode must be one of generate|apply, got '$MODE'" >&2
    exit 1
  fi

  if [[ "$CHART_FLAVOR" != "sonarqube" ]] && [[ "$CHART_FLAVOR" != "sonarqube-dce" ]]; then
    echo "Error: --chart-flavor must be one of sonarqube|sonarqube-dce, got '$CHART_FLAVOR'" >&2
    exit 1
  fi
}

# Checks that the CLI tools this run actually needs are on PATH, so we fail
# fast instead of partway through with a generic "command not found".
check_prerequisites() {
  echo "Checking prerequisites..."

  if ! command -v yq >/dev/null 2>&1; then
    echo "Error: yq (mikefarah/yq, v4+) is required but was not found on PATH" >&2
    exit 1
  fi

  if [[ -z "$VALUES_FILE" ]] || [[ "$MODE" == "apply" ]]; then
    if ! command -v helm >/dev/null 2>&1; then
      echo "Error: helm is required (to read the live release values and/or install Istio) but was not found on PATH" >&2
      exit 1
    fi
  fi

  if [[ "$MODE" == "apply" ]]; then
    if ! command -v kubectl >/dev/null 2>&1; then
      echo "Error: kubectl is required for --mode apply but was not found on PATH" >&2
      exit 1
    fi
  fi

  echo "All required tools found"
  echo ""
}

set_derived_defaults() {
  GATEWAY_NAME="${GATEWAY_NAME:-$RELEASE_NAME}"
  GATEWAY_NAMESPACE="${GATEWAY_NAMESPACE:-$NAMESPACE}"
  CHART_REF="${CHART_REF:-sonarqube/$CHART_FLAVOR}"
  BACKEND_SERVICE="${RELEASE_NAME}-${CHART_FLAVOR}"
  OUTPUT_DIR="${OUTPUT_DIR:-gateway-api-migration-${RELEASE_NAME}}"

  mkdir -p "$OUTPUT_DIR"
  if [[ $? -ne 0 ]]; then
    echo "Error: failed to create output directory '$OUTPUT_DIR'" >&2
    exit 1
  fi

  CURRENT_VALUES_FILE="${OUTPUT_DIR}/current-values-${RELEASE_NAME}.yaml"
  GATEWAY_FILE="${OUTPUT_DIR}/gateway-${RELEASE_NAME}.yaml"
  NEW_VALUES_FILE="${OUTPUT_DIR}/values-gateway-api-${RELEASE_NAME}.yaml"
  NEXT_STEP_CMD="helm upgrade $RELEASE_NAME $CHART_REF -f $NEW_VALUES_FILE -n $NAMESPACE"
}

print_run_summary() {
  echo "=== Ingress-NGINX to Gateway API (Istio) Migration ==="
  echo "Cloud: $CLOUD"
  echo "Mode: $MODE"
  echo "Namespace: $NAMESPACE"
  echo "Release: $RELEASE_NAME"
  echo "Chart flavor: $CHART_FLAVOR (backend service: ${BACKEND_SERVICE}:${BACKEND_PORT})"
  echo "Output directory: $OUTPUT_DIR"
  echo ""
}

# Step 1: load the customer's current Helm values into $CURRENT_VALUES_FILE
load_current_values() {
  echo "Step 1: Loading current Helm values..."

  if [[ -n "$VALUES_FILE" ]]; then
    if [[ ! -f "$VALUES_FILE" ]]; then
      echo "Error: --values-file '$VALUES_FILE' does not exist" >&2
      exit 1
    fi
    cp "$VALUES_FILE" "$CURRENT_VALUES_FILE"
    echo "Using local values file: $VALUES_FILE"
  else
    helm get values "$RELEASE_NAME" -n "$NAMESPACE" -o yaml > "$CURRENT_VALUES_FILE"
    if [[ $? -ne 0 ]] || [[ ! -s "$CURRENT_VALUES_FILE" ]]; then
      echo "Error: failed to read values for release '$RELEASE_NAME' in namespace '$NAMESPACE'" >&2
      rm -f "$CURRENT_VALUES_FILE"
      exit 1
    fi
    echo "Loaded live values for release '$RELEASE_NAME' in namespace '$NAMESPACE'"
  fi

  echo ""
}

# Step 2: detect hostnames/TLS/annotations from the existing ingress config.
# Populates/validates $HOSTNAMES and sets $TLS_ENABLED.
introspect_ingress() {
  echo "Step 2: Introspecting existing ingress configuration..."

  detect_hostnames
  detect_tls
  detect_ingress_nginx_subchart
  check_nginx_annotations
  detect_existing_lb_annotations

  echo ""
}

detect_hostnames() {
  if [[ -z "$HOSTNAMES" ]]; then
    local detected
    detected=$(yq '.ingress.hosts[].name' "$CURRENT_VALUES_FILE" 2>/dev/null | grep -v '^null$' | grep -v '^$' | paste -sd, -)
    if [[ -n "$detected" ]]; then
      HOSTNAMES="$detected"
      echo "Detected hostname(s) from ingress.hosts: $HOSTNAMES"
    else
      echo "Error: no hostnames found in ingress.hosts and --hostnames was not provided" >&2
      exit 1
    fi
  else
    echo "Using hostnames: $HOSTNAMES"
  fi
}

detect_tls() {
  local tls_count
  tls_count=$(yq '.ingress.tls | length' "$CURRENT_VALUES_FILE" 2>/dev/null)
  if [[ "$tls_count" -gt 0 ]] 2>/dev/null; then
    TLS_ENABLED="true"
    echo "Detected TLS termination on the existing ingress (ingress.tls) — the generated Gateway will include an HTTPS listener"
  fi
}

detect_ingress_nginx_subchart() {
  if [[ "$(yq '.["ingress-nginx"].enabled' "$CURRENT_VALUES_FILE" 2>/dev/null)" == "true" ]] || [[ "$(yq '.nginx.enabled' "$CURRENT_VALUES_FILE" 2>/dev/null)" == "true" ]]; then
    echo "Detected the ingress-nginx controller subchart enabled — it will be removed from the generated values.yaml"
  fi
}

# Reads the cloud LoadBalancer Service annotations the customer already has
# configured on the ingress-nginx controller Service (if the bundled subchart
# is used) into $ANNOTATIONS_YAML, so the generated Gateway reuses their
# actual existing configuration instead of a set of assumed defaults.
detect_existing_lb_annotations() {
  local detected
  detected=$(yq '(.["ingress-nginx"].controller.service.annotations // .nginx.controller.service.annotations // {})' "$CURRENT_VALUES_FILE" 2>/dev/null)

  if [[ -z "$detected" ]] || [[ "$detected" == "{}" ]] || [[ "$detected" == "null" ]]; then
    echo "No existing LoadBalancer Service annotations found (checked ingress-nginx.controller.service.annotations / nginx.controller.service.annotations)"
    ANNOTATIONS_YAML="{}"
    return 0
  fi

  echo "Detected existing LoadBalancer Service annotations — reusing them on the generated Gateway:"
  echo "$detected" | sed 's/^/  /'
  ANNOTATIONS_YAML="$detected"
}

# Sets/overrides a single key on a YAML/JSON map passed on stdin, returned on stdout
merge_annotation() {
  local yaml="$1" key="$2" value="$3"
  echo "$yaml" | yq ".[\"$key\"] = \"$value\"" -
}

check_nginx_annotations() {
  echo "Checking nginx.ingress.kubernetes.io/* annotations for Gateway API equivalents..."
  local annotation_keys
  annotation_keys=$(yq '.ingress.annotations | keys | .[]' "$CURRENT_VALUES_FILE" 2>/dev/null | grep -v '^null$')
  if [[ -z "$annotation_keys" ]]; then
    echo "No ingress.annotations found"
    return 0
  fi

  while IFS= read -r key; do
    [[ -z "$key" ]] && continue
    case "$key" in
      "nginx.ingress.kubernetes.io/ssl-redirect")
        echo "  [OK] $key -> covered natively by the Gateway's HTTP+HTTPS listeners, no action needed"
        ;;
      "nginx.ingress.kubernetes.io/rewrite-target"|"nginx.ingress.kubernetes.io/configuration-snippet")
        echo "  [MANUAL] $key -> no automatic equivalent; add a custom httproute.rules entry with a RequestRedirect/URLRewrite filter"
        ;;
      "nginx.ingress.kubernetes.io/proxy-body-size")
        echo "  [MANUAL] $key -> no Gateway API equivalent; requires a follow-up Istio EnvoyFilter"
        ;;
      "nginx.ingress.kubernetes.io/whitelist-source-range")
        echo "  [MANUAL] $key -> no Gateway API equivalent; requires a follow-up Istio AuthorizationPolicy"
        ;;
      *)
        echo "  [UNKNOWN] $key -> not recognized by this script, review manually"
        ;;
    esac
  done <<< "$annotation_keys"
}

# Fails fast (before any file is written) if a cloud-required annotation is
# neither already present on the existing ingress-nginx Service nor supplied
# via a CLI flag. Called directly (not via command substitution) so `exit`
# actually stops the script.
validate_cloud_requirements() {
  if [[ "$CLOUD" == "aws" ]]; then
    local subnets_present
    subnets_present=$(echo "$ANNOTATIONS_YAML" | yq '.["service.beta.kubernetes.io/aws-load-balancer-subnets"] // ""' - 2>/dev/null)
    if [[ -z "$AWS_SUBNETS" ]] && [[ -z "$subnets_present" ]]; then
      echo "Error: no service.beta.kubernetes.io/aws-load-balancer-subnets annotation found on the existing ingress-nginx controller Service, and --aws-subnets was not provided" >&2
      echo "Re-run with --aws-subnets subnet-abc,subnet-def" >&2
      exit 1
    fi
  fi
}

# Builds the cloud-specific annotations block for the Gateway by taking
# whatever LoadBalancer Service annotations the customer already has
# ($ANNOTATIONS_YAML, from detect_existing_lb_annotations) and only
# adding/overriding keys explicitly requested via CLI flags. Never invents
# annotations the customer didn't already have or ask for. May set
# TLS_ENABLED=true (e.g. when an AWS cert ARN is given). Result is echoed.
build_gateway_annotations() {
  local annotations="$ANNOTATIONS_YAML"

  case "$CLOUD" in
    aws)
      [[ -n "$AWS_SUBNETS" ]] && annotations=$(merge_annotation "$annotations" "service.beta.kubernetes.io/aws-load-balancer-subnets" "$AWS_SUBNETS")
      [[ -n "$AWS_SCHEME" ]] && annotations=$(merge_annotation "$annotations" "service.beta.kubernetes.io/aws-load-balancer-scheme" "$AWS_SCHEME")
      if [[ -n "$AWS_CERT_ARN" ]]; then
        annotations=$(merge_annotation "$annotations" "service.beta.kubernetes.io/aws-load-balancer-ssl-cert" "$AWS_CERT_ARN")
        annotations=$(merge_annotation "$annotations" "service.beta.kubernetes.io/aws-load-balancer-ssl-ports" "443")
        TLS_ENABLED="true"
      fi
      ;;
    gcp)
      [[ "$GCP_INTERNAL" == "true" ]] && annotations=$(merge_annotation "$annotations" "networking.gke.io/load-balancer-type" "Internal")
      [[ -n "$GCP_STATIC_IP" ]] && annotations=$(merge_annotation "$annotations" "networking.gke.io/load-balancer-ip-addresses" "$GCP_STATIC_IP")
      ;;
    onprem)
      [[ -n "$METALLB_POOL" ]] && annotations=$(merge_annotation "$annotations" "metallb.io/address-pool" "$METALLB_POOL")
      if [[ -z "$METALLB_POOL" ]] && [[ "$annotations" == "{}" ]]; then
        echo "No --metallb-pool given and no existing LoadBalancer Service annotations detected; the generated Gateway will have no load-balancer annotations. This is expected for NodePort + external hardware LB setups — attach it manually." >&2
      fi
      ;;
  esac

  if [[ "$annotations" == "{}" ]]; then
    echo ""
    return 0
  fi

  echo "$annotations" | yq 'to_entries | .[] | "    " + .key + ": \"" + (.value | tostring) + "\""' - 2>/dev/null
}

write_gateway_manifest() {
  local first_hostname="$1"
  local gateway_annotations="$2"

  {
    echo "apiVersion: gateway.networking.k8s.io/v1"
    echo "kind: Gateway"
    echo "metadata:"
    echo "  name: $GATEWAY_NAME"
    echo "  namespace: $GATEWAY_NAMESPACE"
    if [[ -n "$gateway_annotations" ]]; then
      echo "  annotations:"
      echo "$gateway_annotations"
    fi
    echo "spec:"
    echo "  gatewayClassName: istio"
    echo "  listeners:"
    echo "  - name: http"
    echo "    port: 80"
    echo "    protocol: HTTP"
    echo "    hostname: $first_hostname"
    echo "    allowedRoutes:"
    echo "      namespaces:"
    echo "        from: All"
    if [[ "$TLS_ENABLED" == "true" ]]; then
      echo "  - name: https"
      echo "    port: 443"
      echo "    protocol: HTTP"
      echo "    hostname: $first_hostname"
      echo "    allowedRoutes:"
      echo "      namespaces:"
      echo "        from: All"
    fi
  } > "$GATEWAY_FILE"
}

# Step 3: render the Gateway manifest
render_gateway_manifest() {
  echo "Step 3: Rendering Gateway manifest ($GATEWAY_FILE)..."

  local hostname_list first_hostname gateway_annotations
  hostname_list=$(echo "$HOSTNAMES" | tr ',' '\n')
  first_hostname=$(echo "$hostname_list" | head -n1)

  gateway_annotations=$(build_gateway_annotations)
  write_gateway_manifest "$first_hostname" "$gateway_annotations"

  echo "Gateway manifest written to $GATEWAY_FILE"
  echo ""
}

# Mirrors the chart's "sonarqube.webcontext" helper: env SONAR_WEB_CONTEXT >
# sonarProperties["sonar.web.context"] > sonarWebContext > "", normalized to
# have both a leading and trailing "/".
compute_webcontext() {
  local webcontext
  webcontext=$(yq '.sonarWebContext // ""' "$CURRENT_VALUES_FILE" 2>/dev/null)

  local from_props
  from_props=$(yq '.sonarProperties["sonar.web.context"] // ""' "$CURRENT_VALUES_FILE" 2>/dev/null)
  [[ -n "$from_props" ]] && webcontext="$from_props"

  local from_env
  from_env=$(yq '.env[] | select(.name == "SONAR_WEB_CONTEXT") | .value' "$CURRENT_VALUES_FILE" 2>/dev/null | tail -n1)
  [[ -n "$from_env" ]] && webcontext="$from_env"

  [[ "$webcontext" != /* ]] && webcontext="/${webcontext}"
  [[ "$webcontext" != */ ]] && webcontext="${webcontext}/"

  echo "$webcontext"
}

hostnames_to_yaml_array() {
  local hostname_list="$1"
  local yaml_array="["
  local first="true"

  while IFS= read -r h; do
    [[ -z "$h" ]] && continue
    if [[ "$first" == "true" ]]; then
      yaml_array="${yaml_array}\"$h\""
      first="false"
    else
      yaml_array="${yaml_array}, \"$h\""
    fi
  done <<< "$hostname_list"

  echo "${yaml_array}]"
}

# Step 4: render the replacement values.yaml (ingress/ingress-nginx removed,
# httproute added, everything else preserved from $CURRENT_VALUES_FILE)
render_values_yaml() {
  echo "Step 4: Rendering replacement values.yaml ($NEW_VALUES_FILE)..."

  local hostname_list hostnames_yaml_array webcontext
  hostname_list=$(echo "$HOSTNAMES" | tr ',' '\n')
  hostnames_yaml_array=$(hostnames_to_yaml_array "$hostname_list")
  webcontext=$(compute_webcontext)

  cp "$CURRENT_VALUES_FILE" "$NEW_VALUES_FILE"
  yq -i 'del(.ingress) | del(.["ingress-nginx"]) | del(.nginx)' "$NEW_VALUES_FILE"
  yq -i ".httproute.enabled = true | .httproute.gateway = \"$GATEWAY_NAME\" | .httproute.gatewayNamespace = \"$GATEWAY_NAMESPACE\" | .httproute.hostnames = ${hostnames_yaml_array}" "$NEW_VALUES_FILE"
  yq -i ".httproute.rules = [{\"matches\": [{\"path\": {\"type\": \"PathPrefix\", \"value\": \"$webcontext\"}}], \"backendRefs\": [{\"name\": \"$BACKEND_SERVICE\", \"port\": $BACKEND_PORT}]}]" "$NEW_VALUES_FILE"

  echo "Removed: ingress, ingress-nginx, nginx"
  echo "Added: httproute (enabled, gateway: $GATEWAY_NAME, gatewayNamespace: $GATEWAY_NAMESPACE, hostnames: $HOSTNAMES)"
  echo "Added: httproute.rules with an explicit backendRefs entry targeting ${BACKEND_SERVICE}:${BACKEND_PORT} (path prefix: $webcontext)"
  echo "Everything else from your current values was preserved."
  echo ""
  echo "Note: if any [MANUAL] annotation above needs a custom rule (e.g. a redirect or"
  echo "rewrite filter), add it under httproute.rules in $NEW_VALUES_FILE, alongside the"
  echo "generated backendRefs entry."
  echo ""
}

print_generate_complete() {
  echo "=== Generate Complete ==="
  echo ""
  echo "Files written:"
  echo "  - $GATEWAY_FILE"
  echo "  - $NEW_VALUES_FILE"
  echo ""
  echo "Next steps:"
  echo "  1. Review $GATEWAY_FILE and $NEW_VALUES_FILE"
  echo "  2. Apply the Gateway (kubectl apply -f $GATEWAY_FILE) once Istio is installed,"
  echo "     or re-run this script with --mode apply"
  echo "  3. When ready, run:"
  echo "     $NEXT_STEP_CMD"
}

# Step 5: (apply mode only) install Gateway API CRDs + Istio unless --skip-istio
install_gateway_api_and_istio() {
  echo "Step 5: Installing Gateway API CRDs and Istio..."

  if [[ "$SKIP_ISTIO" == "true" ]]; then
    echo "--skip-istio set, assuming Gateway API CRDs and Istio are already installed"
    echo ""
    return 0
  fi

  echo "Applying Gateway API CRDs..."
  kubectl get crd gateways.gateway.networking.k8s.io >/dev/null 2>&1
  if [[ $? -ne 0 ]]; then
    kubectl apply -f "https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.1.0/standard-install.yaml"
  else
    echo "Gateway API CRDs already present, skipping"
  fi

  helm repo add istio https://istio-release.storage.googleapis.com/charts >/dev/null 2>&1
  helm repo update istio >/dev/null 2>&1

  kubectl create namespace istio-system --dry-run=client -o yaml | kubectl apply -f -

  echo "Installing istio-base..."
  helm upgrade --install istio-base istio/base -n istio-system --wait --timeout=300s
  if [[ $? -ne 0 ]]; then
    echo "Error: failed to install istio-base" >&2
    rm -f "$CURRENT_VALUES_FILE"
    exit 1
  fi

  echo "Installing istiod..."
  helm upgrade --install istiod istio/istiod -n istio-system --wait --timeout=300s
  if [[ $? -ne 0 ]]; then
    echo "Error: failed to install istiod" >&2
    rm -f "$CURRENT_VALUES_FILE"
    exit 1
  fi

  echo ""
}

# Step 6: (apply mode only) apply the generated Gateway manifest
apply_gateway_manifest() {
  echo "Step 6: Applying Gateway manifest..."

  kubectl create namespace "$GATEWAY_NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
  kubectl apply -f "$GATEWAY_FILE"
  if [[ $? -ne 0 ]]; then
    echo "Error: failed to apply $GATEWAY_FILE" >&2
    rm -f "$CURRENT_VALUES_FILE"
    exit 1
  fi
}

print_apply_complete() {
  echo ""
  echo "=== Apply Complete ==="
  echo ""
  echo "Files written:"
  echo "  - $GATEWAY_FILE (applied to the cluster)"
  echo "  - $NEW_VALUES_FILE"
  echo ""
  echo "Next step: when ready, run:"
  echo "$NEXT_STEP_CMD"
}

main() {
  parse_args "$@"
  validate_args
  check_prerequisites
  set_derived_defaults
  trap 'rm -f "$CURRENT_VALUES_FILE"; rmdir "$OUTPUT_DIR" 2>/dev/null' EXIT
  print_run_summary

  load_current_values
  introspect_ingress
  validate_cloud_requirements
  render_gateway_manifest
  render_values_yaml

  if [[ "$MODE" == "generate" ]]; then
    print_generate_complete
    exit 0
  fi

  install_gateway_api_and_istio
  apply_gateway_manifest
  print_apply_complete
}

main "$@"
