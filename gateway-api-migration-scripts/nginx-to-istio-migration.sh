#!/bin/bash

set -euo pipefail

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
      - if the ingress-nginx controller is still enabled, an additional
        "coexistence" values.yaml that keeps ingress-nginx as-is and only adds
        httproute, so both controllers can route traffic in parallel

    This script never runs "helm upgrade" against your SonarQube release. It
    only ever prints the commands for you to run yourself, once you're ready.
    If the ingress-nginx controller is still enabled, two "helm upgrade"
    commands are printed instead of one:
      1. a coexistence upgrade, pinned to --coexist-chart-version (the last
         chart version that still bundles the ingress-nginx subchart — newer
         chart versions drop that subchart entirely regardless of values, so
         staying on the old chart version here is what actually keeps
         ingress-nginx running), using the coexistence values.yaml
      2. once you've verified Istio is handling traffic correctly, the
         cutover upgrade to the target chart version, which removes the
         ingress-nginx controller

USAGE:
    $0 --cloud aws|gcp|onprem [OPTIONS]

OPTIONS:
    --cloud aws|gcp|onprem   Target environment, selects Gateway annotations (REQUIRED)
    --mode generate|apply    generate: render files only (default)
                             apply: also install Gateway API CRDs + Istio (unless
                             --skip-istio) and create the Gateway manifest K8s resource
    --namespace ns           Kubernetes namespace of the release (default: sonarqube)
    --release-name name      Helm release name (default: sonarqube)
    --chart-flavor flavor    sonarqube|sonarqube-dce, used to derive the backend
                             service name (default: sonarqube)
    --chart-ref ref          Chart reference used only in the printed next-step
                             "helm upgrade" command (default: sonarqube/<chart-flavor>)
    --coexist-chart-version  Chart version used only in the printed coexistence-step
                             "helm upgrade" command, when the ingress-nginx controller
                             is still enabled. Must be the last chart version that
                             still bundles the ingress-nginx subchart, since newer
                             chart versions drop it entirely regardless of values
                             (default: 2026.4.0)
    --values-file path       Read values from this local file instead of
                              "helm get values <release> -n <namespace>"
    --hostnames h1,h2        Hostnames for the Gateway/HTTPRoute. Auto-detected
                              from ingress.hosts if omitted
    --gateway-name name      Name for the generated Gateway (default: <release-name>)
    --gateway-namespace ns   Namespace for the generated Gateway (default: <namespace>)
    --output-dir dir         Directory to write generated files into (default:
                              gateway-api-migration-<release-name>)
    --kube-context name      kubeconfig context to use for any live cluster read
                              (helm get values) and, in --mode apply, for
                              installing Istio and applying the Gateway.
                              Defaults to kubectl's current-context. Always
                              printed before use, and confirmed before apply.
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
    --verbose                Show detailed diagnostics for every step, not
                              just the outcome. Failing steps always show
                              their diagnostics regardless of this flag.
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
readonly DEFAULT_CHART_FLAVOR="sonarqube"
readonly MODE_GENERATE="generate"
CLOUD=""
MODE="$MODE_GENERATE"
NAMESPACE="$DEFAULT_CHART_FLAVOR"
RELEASE_NAME="$DEFAULT_CHART_FLAVOR"
CHART_FLAVOR="$DEFAULT_CHART_FLAVOR"
CHART_REF=""
readonly DEFAULT_COEXIST_CHART_VERSION="2026.4.0"
COEXIST_CHART_VERSION="$DEFAULT_COEXIST_CHART_VERSION"
VALUES_FILE=""
HOSTNAMES=""
GATEWAY_NAME=""
GATEWAY_NAMESPACE=""
OUTPUT_DIR=""
KUBE_CONTEXT=""
AWS_SUBNETS=""
AWS_SCHEME=""
AWS_CERT_ARN=""
GCP_INTERNAL="false"
GCP_STATIC_IP=""
METALLB_POOL=""
SKIP_ISTIO="false"
VERBOSE="false"

# Derived, computed once flags are parsed/validated
BACKEND_SERVICE=""
BACKEND_PORT=9000
CURRENT_VALUES_FILE=""
GATEWAY_FILE=""
NEW_VALUES_FILE=""
NEXT_STEP_CMD=""
COEXIST_VALUES_FILE=""
COEXIST_STEP_CMD=""

# Filled in by introspect_ingress
TLS_ENABLED="false"
# "lb": TLS terminated upstream (e.g. AWS NLB via ACM annotation) -- Envoy receives
#       already-decrypted HTTP, so the HTTPS listener must stay protocol HTTP.
# "terminate": TLS terminated by Envoy itself using a k8s Secret (mirrors ingress.tls).
TLS_MODE=""
TLS_SECRET_NAME=""
ANNOTATIONS_YAML="{}"

# Set by detect_ingress_nginx_subchart once the ingress-nginx/nginx subchart
# is known to be enabled in $CURRENT_VALUES_FILE
NGINX_CONTROLLER_PRESENT="false"

# Resolved by resolve_kube_context; empty until then
ACTIVE_KUBE_CONTEXT=""
KUBECTL_CTX_ARGS=()
HELM_CTX_ARGS=()

# Console output helpers ----------------------------------------------------
# Every step prints inside a left-bordered block. Outcome lines (summary/warn)
# are always visible. Diagnostic lines (detail) are buffered and only shown
# immediately with --verbose, or dumped automatically if the step's block
# ends in die() -- so a failure is never missing the context that led to it.
DETAIL_BUFFER=()

box_top() {
  local title="$1"
  printf '┌─ %s\n' "$title"
}

box_line() {
  local line="$1"
  printf '│  %s\n' "$line"
}

box_warn() {
  local message="$1"
  printf '│  ⚠ %s\n' "$message"
}

box_bottom() {
  printf '└─\n'
  echo ""
}

# Outcome line: always visible.
summary() {
  local line="$1"
  box_line "$line"
}

# Actionable/manual-followup line: always visible.
warn() {
  local message="$1"
  box_warn "$message"
}

# Diagnostic line: visible immediately with --verbose, otherwise buffered and
# only surfaced if this step later fails.
detail() {
  local line="$1"
  if [[ "$VERBOSE" == "true" ]]; then
    box_line "$line"
  else
    DETAIL_BUFFER+=("$line")
  fi
}

# Closes a step's block on success, discarding any buffered detail lines.
end_section() {
  DETAIL_BUFFER=()
  box_bottom
}

# Flushes any buffered detail lines, then prints the error and exits. Ensures
# a failure always shows exactly the diagnostics that led to it.
die() {
  local message="$1"
  if [[ ${#DETAIL_BUFFER[@]} -gt 0 ]]; then
    local line
    for line in "${DETAIL_BUFFER[@]}"; do
      box_line "$line"
    done
    DETAIL_BUFFER=()
  fi
  box_warn "Error: $message"
  box_bottom
  exit 1
}

parse_args() {
  local opt
  while [[ $# -gt 0 ]]; do
    opt="$1"
    case "$opt" in
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
      --coexist-chart-version)
        COEXIST_CHART_VERSION="$2"
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
      --kube-context)
        KUBE_CONTEXT="$2"
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
      --verbose)
        VERBOSE="true"
        shift
        ;;
      *)
        echo "Unknown option: $opt" >&2
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

  if [[ "$MODE" != "$MODE_GENERATE" ]] && [[ "$MODE" != "apply" ]]; then
    echo "Error: --mode must be one of generate|apply, got '$MODE'" >&2
    exit 1
  fi

  if [[ "$CHART_FLAVOR" != "$DEFAULT_CHART_FLAVOR" ]] && [[ "$CHART_FLAVOR" != "sonarqube-dce" ]]; then
    echo "Error: --chart-flavor must be one of sonarqube|sonarqube-dce, got '$CHART_FLAVOR'" >&2
    exit 1
  fi
}

# Checks that the CLI tools this run actually needs are on PATH, so we fail
# fast instead of partway through with a generic "command not found".
check_prerequisites() {
  box_top "Checking prerequisites"

  if ! command -v yq >/dev/null 2>&1; then
    die "yq (mikefarah/yq, v4+) is required but was not found on PATH"
  fi

  if { [[ -z "$VALUES_FILE" ]] || [[ "$MODE" == "apply" ]]; } && ! command -v helm >/dev/null 2>&1; then
    die "helm is required (to read the live release values and/or install Istio) but was not found on PATH"
  fi

  if [[ "$MODE" == "apply" ]] && ! command -v kubectl >/dev/null 2>&1; then
    die "kubectl is required for --mode apply but was not found on PATH"
  fi

  summary "All required tools found"
  end_section
}

# Resolves which kube-context this run will use for any live cluster read
# (helm get values) or, in --mode apply, for installing Istio/applying the
# Gateway: either the explicit --kube-context, or kubectl's current-context.
# Always printed so the operator can catch a wrong-cluster kubeconfig before
# anything happens. Skipped entirely when nothing will touch a cluster (i.e.
# --values-file was given and --mode generate).
resolve_kube_context() {
  if [[ -n "$VALUES_FILE" ]] && [[ "$MODE" == "$MODE_GENERATE" ]]; then
    return 0
  fi

  box_top "Resolving kube-context"

  if [[ -n "$KUBE_CONTEXT" ]]; then
    ACTIVE_KUBE_CONTEXT="$KUBE_CONTEXT"
  else
    ACTIVE_KUBE_CONTEXT=$(kubectl config current-context 2>/dev/null)
  fi

  if [[ -z "$ACTIVE_KUBE_CONTEXT" ]]; then
    die "could not resolve a kube-context (no --kube-context given and no current-context set in kubeconfig)"
  fi

  KUBECTL_CTX_ARGS=(--context "$ACTIVE_KUBE_CONTEXT")
  HELM_CTX_ARGS=(--kube-context "$ACTIVE_KUBE_CONTEXT")

  summary "Using kube-context: $ACTIVE_KUBE_CONTEXT"
  end_section
}

# Fails fast unless the operator explicitly confirms the resolved kube-context
# is the intended cluster, before Istio is installed/upgraded or the Gateway
# is applied.
confirm_apply_target() {
  local reply
  echo "About to install/upgrade Istio and apply the Gateway resource to your Kubernetes cluster (kube-context '$ACTIVE_KUBE_CONTEXT', namespace '$GATEWAY_NAMESPACE')."
  read -r -p "Continue? [y/N] " reply
  if [[ ! "$reply" =~ ^[Yy]$ ]]; then
    echo "Aborted." >&2
    exit 1
  fi
  echo ""
}

set_derived_defaults() {
  GATEWAY_NAME="${GATEWAY_NAME:-$RELEASE_NAME}"
  GATEWAY_NAMESPACE="${GATEWAY_NAMESPACE:-$NAMESPACE}"
  CHART_REF="${CHART_REF:-sonarqube/$CHART_FLAVOR}"
  BACKEND_SERVICE="${RELEASE_NAME}-${CHART_FLAVOR}"
  OUTPUT_DIR="${OUTPUT_DIR:-gateway-api-migration-${RELEASE_NAME}}"

  if ! mkdir -p "$OUTPUT_DIR"; then
    echo "Error: failed to create output directory '$OUTPUT_DIR'" >&2
    exit 1
  fi

  CURRENT_VALUES_FILE="${OUTPUT_DIR}/current-values-${RELEASE_NAME}.yaml"
  GATEWAY_FILE="${OUTPUT_DIR}/gateway-${RELEASE_NAME}.yaml"
  NEW_VALUES_FILE="${OUTPUT_DIR}/values-gateway-api-${RELEASE_NAME}.yaml"
  NEXT_STEP_CMD="helm upgrade $RELEASE_NAME $CHART_REF -f $NEW_VALUES_FILE -n $NAMESPACE"
  COEXIST_VALUES_FILE="${OUTPUT_DIR}/values-coexist-${RELEASE_NAME}.yaml"
  COEXIST_STEP_CMD="helm upgrade $RELEASE_NAME $CHART_REF --version $COEXIST_CHART_VERSION -f $COEXIST_VALUES_FILE -n $NAMESPACE"
}

print_run_summary() {
  box_top "Ingress-NGINX to Gateway API (Istio) Migration"
  summary "Cloud: $CLOUD"
  summary "Mode: $MODE"
  summary "Namespace: $NAMESPACE"
  summary "Release: $RELEASE_NAME"
  summary "Chart flavor: $CHART_FLAVOR (backend service: ${BACKEND_SERVICE}:${BACKEND_PORT})"
  summary "Output directory: $OUTPUT_DIR"
  end_section
}

# Step 1: load the customer's current Helm values into $CURRENT_VALUES_FILE
load_current_values() {
  box_top "Step 1: Loading current Helm values"

  if [[ -n "$VALUES_FILE" ]]; then
    if [[ ! -f "$VALUES_FILE" ]]; then
      die "--values-file '$VALUES_FILE' does not exist"
    fi
    cp "$VALUES_FILE" "$CURRENT_VALUES_FILE"
    summary "Using local values file: $VALUES_FILE"
  else
    if ! helm "${HELM_CTX_ARGS[@]}" get values "$RELEASE_NAME" -n "$NAMESPACE" -o yaml > "$CURRENT_VALUES_FILE" || [[ ! -s "$CURRENT_VALUES_FILE" ]]; then
      rm -f "$CURRENT_VALUES_FILE"
      die "failed to read values for release '$RELEASE_NAME' in namespace '$NAMESPACE'"
    fi
    summary "Loaded live values for release '$RELEASE_NAME' in namespace '$NAMESPACE'"
  fi

  end_section
}

# Detects a release that's already fully on Gateway API (httproute.enabled, and
# no ingress/ingress-nginx config left to remove) so re-running this script is a
# clean no-op instead of failing later on missing ingress.hosts/annotations.
check_already_migrated() {
  local httproute_enabled ingress_enabled ingress_nginx_enabled nginx_enabled

  httproute_enabled=$(yq '.httproute.enabled // false' "$CURRENT_VALUES_FILE" 2>/dev/null)
  ingress_enabled=$(yq '.ingress.enabled // false' "$CURRENT_VALUES_FILE" 2>/dev/null)
  ingress_nginx_enabled=$(yq '.["ingress-nginx"].enabled // false' "$CURRENT_VALUES_FILE" 2>/dev/null)
  nginx_enabled=$(yq '.nginx.enabled // false' "$CURRENT_VALUES_FILE" 2>/dev/null)

  if [[ "$httproute_enabled" == "true" ]] \
    && [[ "$ingress_enabled" != "true" ]] \
    && [[ "$ingress_nginx_enabled" != "true" ]] \
    && [[ "$nginx_enabled" != "true" ]]; then
    box_top "Already migrated"
    summary "This release already has httproute.enabled=true and no ingress/ingress-nginx config left — it looks like the migration was already applied. Nothing to do."
    end_section
    exit 0
  fi
}

# Refines $BACKEND_SERVICE (initially "<release-name>-<chart-flavor>", set in
# set_derived_defaults) using any nameOverride/fullnameOverride found in the
# loaded values, mirroring the chart's own "sonarqube.fullname" helper:
# fullnameOverride wins outright; otherwise "<release-name>-<nameOverride or
# chart-flavor>". Without this, an override on the live release would make the
# generated backendRefs point at a Service that doesn't exist.
resolve_backend_service() {
  local fullname_override
  fullname_override=$(yq '.fullnameOverride // ""' "$CURRENT_VALUES_FILE" 2>/dev/null)
  if [[ -n "$fullname_override" ]] && [[ "$fullname_override" != "null" ]]; then
    BACKEND_SERVICE="${fullname_override:0:63}"
    box_top "Resolving backend service name"
    summary "Detected fullnameOverride: $BACKEND_SERVICE -- using it as the backendRefs service name"
    end_section
    return 0
  fi

  local name_override
  name_override=$(yq '.nameOverride // ""' "$CURRENT_VALUES_FILE" 2>/dev/null)
  if [[ -n "$name_override" ]] && [[ "$name_override" != "null" ]]; then
    BACKEND_SERVICE="${RELEASE_NAME}-${name_override}"
    BACKEND_SERVICE="${BACKEND_SERVICE:0:63}"
    box_top "Resolving backend service name"
    summary "Detected nameOverride: $name_override -- backend service resolved to $BACKEND_SERVICE"
    end_section
  fi
}

# Step 2: detect hostnames/TLS/annotations from the existing ingress config.
# Populates/validates $HOSTNAMES and sets $TLS_ENABLED.
introspect_ingress() {
  box_top "Step 2: Introspecting existing ingress configuration"

  detect_hostnames
  detect_existing_lb_annotations
  detect_tls
  detect_ingress_nginx_subchart
  check_nginx_annotations
  check_ingress_nginx_controller_properties

  end_section
}

detect_hostnames() {
  if [[ -z "$HOSTNAMES" ]]; then
    local detected
    detected=$(yq '.ingress.hosts[].name' "$CURRENT_VALUES_FILE" 2>/dev/null | grep -v '^null$' | grep -v '^$' | paste -sd, - || true)
    if [[ -n "$detected" ]]; then
      HOSTNAMES="$detected"
      summary "Detected hostname(s) from ingress.hosts: $HOSTNAMES"
    else
      die "no hostnames found in ingress.hosts and --hostnames was not provided"
    fi
  else
    summary "Using hostnames: $HOSTNAMES"
  fi
}

detect_tls() {
  if [[ "$CLOUD" == "aws" ]]; then
    if [[ -n "$AWS_CERT_ARN" ]]; then
      TLS_ENABLED="true"
      TLS_MODE="lb"
      summary "--aws-cert-arn given — TLS will be terminated at the NLB using that ACM cert, not by the ingress controller pod. The generated Gateway's HTTPS listener will stay protocol HTTP since Envoy receives already-decrypted traffic."
      return 0
    fi

    local ssl_cert_present
    ssl_cert_present=$(echo "$ANNOTATIONS_YAML" | yq '.["service.beta.kubernetes.io/aws-load-balancer-ssl-cert"] // ""' - 2>/dev/null)
    if [[ -n "$ssl_cert_present" ]] && [[ "$ssl_cert_present" != "null" ]]; then
      TLS_ENABLED="true"
      TLS_MODE="lb"
      summary "Detected an existing aws-load-balancer-ssl-cert annotation — TLS is terminated at the NLB using an ACM cert, not by the ingress controller pod. The generated Gateway's HTTPS listener will stay protocol HTTP since Envoy receives already-decrypted traffic."
      return 0
    fi
  fi

  local tls_count
  tls_count=$(yq '.ingress.tls | length' "$CURRENT_VALUES_FILE" 2>/dev/null)
  if [[ "$tls_count" -gt 0 ]] 2>/dev/null; then
    TLS_ENABLED="true"
    TLS_MODE="terminate"
    TLS_SECRET_NAME=$(yq '.ingress.tls[0].secretName // ""' "$CURRENT_VALUES_FILE" 2>/dev/null)
    if [[ -n "$TLS_SECRET_NAME" ]] && [[ "$TLS_SECRET_NAME" != "null" ]]; then
      summary "Detected TLS termination via ingress.tls (secretName: $TLS_SECRET_NAME) — the generated Gateway will include an HTTPS listener terminating TLS using that Secret"
    else
      TLS_SECRET_NAME=""
      warn "Detected ingress.tls entries but no secretName set — the generated Gateway will include an HTTPS listener, but you must manually set tls.certificateRefs"
    fi
    return 0
  fi

  local target_port_https
  target_port_https=$(yq '(.["ingress-nginx"].controller.service.targetPorts.https // .nginx.controller.service.targetPorts.https // "") | tostring' "$CURRENT_VALUES_FILE" 2>/dev/null)
  if [[ -n "$target_port_https" ]] && [[ "$target_port_https" != "null" ]] && [[ "$target_port_https" != "https" ]]; then
    TLS_ENABLED="true"
    TLS_MODE="lb"
    summary "Detected ingress-nginx.controller.service.targetPorts.https: $target_port_https (not \"https\") — TLS is terminated upstream of the ingress controller pod, so the generated Gateway's HTTPS listener will stay protocol HTTP since Envoy receives already-decrypted traffic."
  fi
}

detect_ingress_nginx_subchart() {
  if [[ "$(yq '.["ingress-nginx"].enabled' "$CURRENT_VALUES_FILE" 2>/dev/null)" == "true" ]] || [[ "$(yq '.nginx.enabled' "$CURRENT_VALUES_FILE" 2>/dev/null)" == "true" ]]; then
    NGINX_CONTROLLER_PRESENT="true"
    summary "Detected the ingress-nginx controller subchart enabled — it will be removed from the generated values.yaml"
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
    detail "No existing LoadBalancer Service annotations found (checked ingress-nginx.controller.service.annotations / nginx.controller.service.annotations)"
    ANNOTATIONS_YAML="{}"
    return 0
  fi

  summary "Detected existing LoadBalancer Service annotations — reusing them on the generated Gateway"
  local line
  while IFS= read -r line; do
    [[ -z "$line" ]] && continue
    detail "  $line"
  done <<< "$detected"
  ANNOTATIONS_YAML="$detected"
}

# Sets/overrides a single key on a YAML/JSON map passed on stdin, returned on stdout.
# Logs (to stderr, so it never pollutes the returned YAML) whether this is adding a
# new annotation or overriding one already detected on the existing Service, and which
# flag triggered it. Runs inside a command substitution ($(...)) so it must not use the
# box_*/summary/detail helpers, which write to stdout and would corrupt the return value.
merge_annotation() {
  local yaml="$1" key="$2" value="$3" flag="$4"
  local existing
  existing=$(echo "$yaml" | yq ".[\"$key\"] // \"\"" -)
  if [[ -n "$existing" ]] && [[ "$existing" != "null" ]]; then
    echo "│  ⚠ [$flag] overriding existing annotation $key (was \"$existing\") -> \"$value\"" >&2
  else
    echo "│  [$flag] adding annotation $key -> \"$value\"" >&2
  fi
  echo "$yaml" | yq ".[\"$key\"] = \"$value\"" -
}

check_nginx_annotations() {
  local annotation_keys
  annotation_keys=$(yq '.ingress.annotations | keys | .[]' "$CURRENT_VALUES_FILE" 2>/dev/null | grep -v '^null$' || true)
  if [[ -z "$annotation_keys" ]]; then
    detail "No ingress.annotations found"
    return 0
  fi

  while IFS= read -r key; do
    [[ -z "$key" ]] && continue
    case "$key" in
      "nginx.ingress.kubernetes.io/ssl-redirect")
        detail "[OK] $key -> covered natively by the Gateway's HTTP+HTTPS listeners, no action needed"
        ;;
      "nginx.ingress.kubernetes.io/rewrite-target"|"nginx.ingress.kubernetes.io/configuration-snippet")
        warn "$key -> no automatic equivalent; add a custom httproute.rules entry with a RequestRedirect/URLRewrite filter"
        ;;
      "nginx.ingress.kubernetes.io/proxy-body-size")
        warn "$key -> no Gateway API equivalent; requires a follow-up Istio EnvoyFilter"
        ;;
      "nginx.ingress.kubernetes.io/whitelist-source-range")
        warn "$key -> no Gateway API equivalent; requires a follow-up Istio AuthorizationPolicy"
        ;;
      *)
        warn "$key -> not recognized by this script, review manually"
        ;;
    esac
  done <<< "$annotation_keys"
}

# Flags commonly-set ingress-nginx.controller.* properties (beyond the
# service.annotations already reused verbatim, and targetPorts.https already
# handled in detect_tls) that have no automatic Gateway API/Istio equivalent.
# Informational only, like check_nginx_annotations -- never written to the
# generated files.
check_ingress_nginx_controller_properties() {
  local found="false"

  local source_ranges
  source_ranges=$(yq '(.["ingress-nginx"].controller.service.loadBalancerSourceRanges // .nginx.controller.service.loadBalancerSourceRanges // []) | length' "$CURRENT_VALUES_FILE" 2>/dev/null)
  if [[ "$source_ranges" -gt 0 ]] 2>/dev/null; then
    found="true"
    warn "controller.service.loadBalancerSourceRanges -> no Gateway API equivalent; restrict source IPs with an Istio AuthorizationPolicy, or an externalTrafficPolicy/LB-level rule"
  fi

  local traffic_policy
  traffic_policy=$(yq '.["ingress-nginx"].controller.service.externalTrafficPolicy // .nginx.controller.service.externalTrafficPolicy // ""' "$CURRENT_VALUES_FILE" 2>/dev/null)
  if [[ -n "$traffic_policy" ]] && [[ "$traffic_policy" != "null" ]]; then
    found="true"
    warn "controller.service.externalTrafficPolicy: $traffic_policy -> not carried over automatically; set the same value on the istio-ingressgateway Service if client source IP preservation matters"
  fi

  local config_keys
  config_keys=$(yq '(.["ingress-nginx"].controller.config // .nginx.controller.config // {}) | keys | .[]' "$CURRENT_VALUES_FILE" 2>/dev/null | grep -v '^null$' || true)
  if [[ -n "$config_keys" ]]; then
    found="true"
    while IFS= read -r key; do
      [[ -z "$key" ]] && continue
      warn "controller.config.$key -> global nginx ConfigMap setting, not covered by this script; check whether an Istio equivalent (EnvoyFilter, mesh config, or per-route Gateway API filter) is needed"
    done <<< "$config_keys"
  fi

  if [[ "$found" == "false" ]]; then
    detail "No loadBalancerSourceRanges, externalTrafficPolicy, or controller.config overrides found"
  fi
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
      box_top "Validating cloud requirements"
      die "no service.beta.kubernetes.io/aws-load-balancer-subnets annotation found on the existing ingress-nginx controller Service, and --aws-subnets was not provided. Re-run with --aws-subnets subnet-abc,subnet-def"
    fi
  fi
}

# Builds the cloud-specific annotations block for the Gateway by taking
# whatever LoadBalancer Service annotations the customer already has
# ($ANNOTATIONS_YAML, from detect_existing_lb_annotations) and only
# adding/overriding keys explicitly requested via CLI flags. Never invents
# annotations the customer didn't already have or ask for. Must not set
# TLS_ENABLED/TLS_MODE here — this runs via command substitution, so any
# variable assignment happens in a subshell and is lost; TLS state is
# decided upfront in detect_tls instead. Result is echoed.
build_gateway_annotations() {
  local annotations="$ANNOTATIONS_YAML"

  case "$CLOUD" in
    aws)
      [[ -n "$AWS_SUBNETS" ]] && annotations=$(merge_annotation "$annotations" "service.beta.kubernetes.io/aws-load-balancer-subnets" "$AWS_SUBNETS" "--aws-subnets")
      [[ -n "$AWS_SCHEME" ]] && annotations=$(merge_annotation "$annotations" "service.beta.kubernetes.io/aws-load-balancer-scheme" "$AWS_SCHEME" "--aws-scheme")
      if [[ -n "$AWS_CERT_ARN" ]]; then
        annotations=$(merge_annotation "$annotations" "service.beta.kubernetes.io/aws-load-balancer-ssl-cert" "$AWS_CERT_ARN" "--aws-cert-arn")
        annotations=$(merge_annotation "$annotations" "service.beta.kubernetes.io/aws-load-balancer-ssl-ports" "443" "--aws-cert-arn")
      fi
      ;;
    gcp)
      [[ "$GCP_INTERNAL" == "true" ]] && annotations=$(merge_annotation "$annotations" "networking.gke.io/load-balancer-type" "Internal" "--gcp-internal")
      [[ -n "$GCP_STATIC_IP" ]] && annotations=$(merge_annotation "$annotations" "networking.gke.io/load-balancer-ip-addresses" "$GCP_STATIC_IP" "--gcp-static-ip")
      ;;
    onprem)
      [[ -n "$METALLB_POOL" ]] && annotations=$(merge_annotation "$annotations" "metallb.io/address-pool" "$METALLB_POOL" "--metallb-pool")
      if [[ -z "$METALLB_POOL" ]] && [[ "$annotations" == "{}" ]]; then
        echo "│  ⚠ No --metallb-pool given and no existing LoadBalancer Service annotations detected; the generated Gateway will have no load-balancer annotations. This is expected for NodePort + external hardware LB setups — attach it manually." >&2
      fi
      ;;
    *)
      echo "Error: unexpected --cloud value '$CLOUD' (validate_args should have rejected this)" >&2
      exit 1
      ;;
  esac

  if [[ "$annotations" == "{}" ]]; then
    echo ""
    return 0
  fi

  echo "$annotations" | yq 'to_entries | .[] | "    " + .key + ": \"" + (.value | tostring) + "\""' - 2>/dev/null
}

# Listeners intentionally omit "hostname:" (matching all hosts) rather than
# pinning to a single hostname: a Gateway API HTTPRoute can only attach to a
# listener if its hostnames intersect, and the generated HTTPRoute carries
# every configured hostname (see render_values_yaml) — pinning the listener
# to one hostname would silently stop routing the others.
write_gateway_manifest() {
  local gateway_annotations="$1"

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
    echo "    allowedRoutes:"
    echo "      namespaces:"
    echo "        from: All"
    if [[ "$TLS_ENABLED" == "true" ]]; then
      echo "  - name: https"
      echo "    port: 443"
      if [[ "$TLS_MODE" == "lb" ]]; then
        echo "    protocol: HTTP"
      else
        echo "    protocol: HTTPS"
      fi
      echo "    allowedRoutes:"
      echo "      namespaces:"
      echo "        from: All"
      if [[ "$TLS_MODE" == "terminate" ]]; then
        echo "    tls:"
        echo "      mode: Terminate"
        echo "      certificateRefs:"
        if [[ -n "$TLS_SECRET_NAME" ]]; then
          echo "      - name: $TLS_SECRET_NAME"
          echo "        kind: Secret"
        else
          echo "      - name: REPLACE_ME_TLS_SECRET_NAME # TODO: no ingress.tls.secretName detected; set to the Secret holding your TLS cert"
          echo "        kind: Secret"
        fi
        # The Secret lives in the app namespace ($NAMESPACE), detected from
        # ingress.tls.secretName. If the Gateway lives elsewhere
        # (--gateway-namespace), it must reference the Secret across
        # namespaces explicitly, which also requires a ReferenceGrant in the
        # Secret's namespace (see write_tls_reference_grant).
        if [[ "$GATEWAY_NAMESPACE" != "$NAMESPACE" ]]; then
          echo "        namespace: $NAMESPACE"
        fi
      fi
    fi
  } > "$GATEWAY_FILE"

  if [[ "$TLS_MODE" == "terminate" ]] && [[ "$GATEWAY_NAMESPACE" != "$NAMESPACE" ]]; then
    write_tls_reference_grant
  fi
}

# Appends a ReferenceGrant (to $GATEWAY_FILE, as a second YAML document) that
# permits the Gateway in $GATEWAY_NAMESPACE to reference the TLS Secret sitting
# in $NAMESPACE. Without this, cross-namespace certificateRefs are rejected by
# the Gateway API implementation regardless of the explicit "namespace:" field.
write_tls_reference_grant() {
  local secret_name="${TLS_SECRET_NAME:-REPLACE_ME_TLS_SECRET_NAME}"

  summary "Cross-namespace TLS detected (Gateway in $GATEWAY_NAMESPACE, Secret in $NAMESPACE) — appending a ReferenceGrant in $NAMESPACE to $GATEWAY_FILE"

  {
    echo "---"
    echo "apiVersion: gateway.networking.k8s.io/v1beta1"
    echo "kind: ReferenceGrant"
    echo "metadata:"
    echo "  name: ${GATEWAY_NAME}-tls-secret"
    echo "  namespace: $NAMESPACE"
    echo "spec:"
    echo "  from:"
    echo "  - group: gateway.networking.k8s.io"
    echo "    kind: Gateway"
    echo "    namespace: $GATEWAY_NAMESPACE"
    echo "  to:"
    echo "  - group: \"\""
    echo "    kind: Secret"
    echo "    name: $secret_name"
  } >> "$GATEWAY_FILE"
}

# Step 3: render the Gateway manifest
render_gateway_manifest() {
  box_top "Step 3: Rendering Gateway manifest"

  local gateway_annotations
  gateway_annotations=$(build_gateway_annotations)
  write_gateway_manifest "$gateway_annotations"

  summary "Written to $GATEWAY_FILE"
  end_section
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

# Adds httproute (enabled + gateway/hostnames/rules) to the values file at
# $1, using the hostnames/webcontext computed by the caller. Shared by the
# coexistence values (ingress-nginx left untouched) and the final cutover
# values (ingress/ingress-nginx/nginx removed).
add_httproute_to_values() {
  local target_file="$1" hostnames_yaml_array="$2" webcontext="$3"

  yq -i ".httproute.enabled = true | .httproute.gateway = \"$GATEWAY_NAME\" | .httproute.gatewayNamespace = \"$GATEWAY_NAMESPACE\" | .httproute.hostnames = ${hostnames_yaml_array}" "$target_file"
  yq -i ".httproute.rules = [{\"matches\": [{\"path\": {\"type\": \"PathPrefix\", \"value\": \"$webcontext\"}}], \"backendRefs\": [{\"name\": \"$BACKEND_SERVICE\", \"port\": $BACKEND_PORT}]}]" "$target_file"
}

# Step 4: render the replacement values.yaml (ingress/ingress-nginx removed,
# httproute added, everything else preserved from $CURRENT_VALUES_FILE). If
# ingress-nginx is still enabled, also renders a coexistence values.yaml that
# leaves ingress-nginx running and only adds httproute, so both controllers
# can route traffic in parallel while Istio is verified -- the cutover
# values.yaml above must be applied on a chart version that still bundles
# ingress-nginx (see --coexist-chart-version) for the coexistence to actually
# hold, since newer chart versions drop that subchart regardless of values.
render_values_yaml() {
  box_top "Step 4: Rendering replacement values.yaml"

  local hostname_list hostnames_yaml_array webcontext
  hostname_list=$(echo "$HOSTNAMES" | tr ',' '\n')
  hostnames_yaml_array=$(hostnames_to_yaml_array "$hostname_list")
  webcontext=$(compute_webcontext)

  if [[ "$NGINX_CONTROLLER_PRESENT" == "true" ]]; then
    cp "$CURRENT_VALUES_FILE" "$COEXIST_VALUES_FILE"
    add_httproute_to_values "$COEXIST_VALUES_FILE" "$hostnames_yaml_array" "$webcontext"
    summary "Written to $COEXIST_VALUES_FILE (ingress-nginx left enabled, httproute added -- for the coexistence step, applied with --coexist-chart-version $COEXIST_CHART_VERSION)"
  fi

  cp "$CURRENT_VALUES_FILE" "$NEW_VALUES_FILE"
  yq -i 'del(.ingress) | del(.["ingress-nginx"]) | del(.nginx)' "$NEW_VALUES_FILE"
  add_httproute_to_values "$NEW_VALUES_FILE" "$hostnames_yaml_array" "$webcontext"

  summary "Written to $NEW_VALUES_FILE"
  detail "Removed: ingress, ingress-nginx, nginx"
  detail "Added: httproute (enabled, gateway: $GATEWAY_NAME, gatewayNamespace: $GATEWAY_NAMESPACE, hostnames: $HOSTNAMES)"
  detail "Added: httproute.rules with an explicit backendRefs entry targeting ${BACKEND_SERVICE}:${BACKEND_PORT} (path prefix: $webcontext)"
  detail "Everything else from your current values was preserved"
  warn "if any [MANUAL] annotation flagged above needs a custom rule (e.g. a redirect or rewrite filter), add it under httproute.rules in $NEW_VALUES_FILE, alongside the generated backendRefs entry"
  end_section
}

print_generate_complete() {
  box_top "Generate Complete"
  summary "Files written:"
  summary "  - $GATEWAY_FILE"
  if [[ "$NGINX_CONTROLLER_PRESENT" == "true" ]]; then
    summary "  - $COEXIST_VALUES_FILE"
  fi
  summary "  - $NEW_VALUES_FILE"
  summary ""
  summary "Next steps:"
  summary ""
  if [[ "$NGINX_CONTROLLER_PRESENT" == "true" ]]; then
    summary "  1. Review $GATEWAY_FILE, $COEXIST_VALUES_FILE, and $NEW_VALUES_FILE"
    summary ""
    summary "  2. Apply the Gateway (kubectl apply -f $GATEWAY_FILE) once Istio is installed,"
    summary "     or re-run this script with --mode apply. Keep $GATEWAY_FILE around afterwards"
    summary "     (e.g. commit it to version control) -- you'll need it to re-apply the Gateway"
    summary "     to this or another cluster later"
    summary ""
    summary "  3. Coexistence step -- run this on the last chart version that still bundles"
    summary "     ingress-nginx ($COEXIST_CHART_VERSION), so ingress-nginx keeps running"
    summary "     alongside Istio instead of being pruned by a newer chart:"
    summary "     $COEXIST_STEP_CMD"
    summary ""
    summary "  4. Verify Istio is handling traffic correctly, e.g. port-forward the Istio ingress"
    summary "     gateway Service (kubectl port-forward -n istio-system svc/istio-ingressgateway"
    summary "     8080:80) and curl it with the Host header set to your hostname -- or use"
    summary "     whatever else fits your setup (its LoadBalancer URL, a temporary DNS record, etc.)"
    summary ""
    summary "  5. Cutover step -- once verified, run this to move to the target chart"
    summary "     version and remove the ingress-nginx controller:"
    summary "     $NEXT_STEP_CMD"
  else
    summary "  1. Review $GATEWAY_FILE and $NEW_VALUES_FILE"
    summary ""
    summary "  2. Apply the Gateway (kubectl apply -f $GATEWAY_FILE) once Istio is installed,"
    summary "     or re-run this script with --mode apply. Keep $GATEWAY_FILE around afterwards"
    summary "     (e.g. commit it to version control) -- you'll need it to re-apply the Gateway"
    summary "     to this or another cluster later"
    summary ""
    summary "  3. Verify Istio is handling traffic correctly, e.g. port-forward the Istio ingress"
    summary "     gateway Service (kubectl port-forward -n istio-system svc/istio-ingressgateway"
    summary "     8080:80) and curl it with the Host header set to your hostname -- or use"
    summary "     whatever else fits your setup (its LoadBalancer URL, a temporary DNS record, etc.)"
    summary ""
    summary "  4. When ready, run:"
    summary "     $NEXT_STEP_CMD"
  fi
  summary ""
  warn "Please update your source-controlled values.yaml with the contents of $NEW_VALUES_FILE, so future 'helm upgrade' runs keep using it."
  warn "Please also keep $GATEWAY_FILE under version control -- it's not managed by helm upgrade, so it's the only record of the Gateway you'll need to re-apply it later (this or another cluster, disaster recovery, etc.)"
  summary ""
  end_section
}

# Step 5: (apply mode only) install Gateway API CRDs + Istio unless --skip-istio
install_gateway_api_and_istio() {
  box_top "Step 5: Installing Gateway API CRDs and Istio"

  if [[ "$SKIP_ISTIO" == "true" ]]; then
    summary "--skip-istio set, assuming Gateway API CRDs and Istio are already installed"
    end_section
    return 0
  fi

  detail "Applying Gateway API CRDs..."
  if ! kubectl "${KUBECTL_CTX_ARGS[@]}" get crd gateways.gateway.networking.k8s.io >/dev/null 2>&1; then
    kubectl "${KUBECTL_CTX_ARGS[@]}" apply -f "https://github.com/kubernetes-sigs/gateway-api/releases/latest/download/standard-install.yaml"
  else
    detail "Gateway API CRDs already present, skipping"
  fi

  helm repo add istio https://istio-release.storage.googleapis.com/charts >/dev/null 2>&1
  helm repo update istio >/dev/null 2>&1

  kubectl "${KUBECTL_CTX_ARGS[@]}" create namespace istio-system --dry-run=client -o yaml | kubectl "${KUBECTL_CTX_ARGS[@]}" apply -f -

  detail "Installing istio-base..."
  if ! helm "${HELM_CTX_ARGS[@]}" upgrade --install istio-base istio/base -n istio-system --wait --timeout=300s; then
    rm -f "$CURRENT_VALUES_FILE"
    die "failed to install istio-base"
  fi

  detail "Installing istiod..."
  if ! helm "${HELM_CTX_ARGS[@]}" upgrade --install istiod istio/istiod -n istio-system --wait --timeout=300s; then
    rm -f "$CURRENT_VALUES_FILE"
    die "failed to install istiod"
  fi

  summary "Gateway API CRDs and Istio (base + istiod) installed"
  end_section
}

# Step 6: (apply mode only) apply the generated Gateway manifest
apply_gateway_manifest() {
  box_top "Step 6: Applying Gateway manifest"

  kubectl "${KUBECTL_CTX_ARGS[@]}" create namespace "$GATEWAY_NAMESPACE" --dry-run=client -o yaml | kubectl "${KUBECTL_CTX_ARGS[@]}" apply -f -
  if ! kubectl "${KUBECTL_CTX_ARGS[@]}" apply -f "$GATEWAY_FILE"; then
    rm -f "$CURRENT_VALUES_FILE"
    die "failed to apply $GATEWAY_FILE"
  fi

  summary "Applied $GATEWAY_FILE"
  end_section
}

print_apply_complete() {
  box_top "Apply Complete"
  summary "Files written:"
  summary "  - $GATEWAY_FILE (applied to the cluster)"
  if [[ "$NGINX_CONTROLLER_PRESENT" == "true" ]]; then
    summary "  - $COEXIST_VALUES_FILE"
  fi
  summary "  - $NEW_VALUES_FILE"
  summary ""
  if [[ "$NGINX_CONTROLLER_PRESENT" == "true" ]]; then
    summary "Coexistence step -- run this on the last chart version that still bundles"
    summary "ingress-nginx ($COEXIST_CHART_VERSION), so ingress-nginx keeps running"
    summary "alongside Istio instead of being pruned by a newer chart:"
    summary "$COEXIST_STEP_CMD"
    summary ""
  fi
  summary "Verify Istio is handling traffic correctly, e.g. port-forward the Istio ingress"
  summary "gateway Service (kubectl port-forward -n istio-system svc/istio-ingressgateway"
  summary "8080:80) and curl it with the Host header set to your hostname -- or use whatever"
  summary "else fits your setup (its LoadBalancer URL, a temporary DNS record, etc.)"
  summary ""
  if [[ "$NGINX_CONTROLLER_PRESENT" == "true" ]]; then
    summary "Once verified, run the cutover step below to move to the target chart"
    summary "version and remove the ingress-nginx controller:"
  else
    summary "Next step: once verified, run:"
  fi
  summary "$NEXT_STEP_CMD"
  warn "Please update your source-controlled values.yaml with the contents of $NEW_VALUES_FILE, so future 'helm upgrade' runs keep using it."
  warn "Please also keep $GATEWAY_FILE under version control -- it's not managed by helm upgrade, so it's the only record of the Gateway you'll need to re-apply it later (this or another cluster, disaster recovery, etc.)"
  end_section
}

main() {
  parse_args "$@"
  validate_args
  check_prerequisites
  set_derived_defaults
  trap 'ec=$?; rm -f "$CURRENT_VALUES_FILE"; rmdir "$OUTPUT_DIR" 2>/dev/null || true; exit $ec' EXIT
  print_run_summary
  resolve_kube_context

  load_current_values
  check_already_migrated
  resolve_backend_service
  introspect_ingress
  validate_cloud_requirements
  render_gateway_manifest
  render_values_yaml

  if [[ "$MODE" == "$MODE_GENERATE" ]]; then
    print_generate_complete
    exit 0
  fi

  confirm_apply_target
  install_gateway_api_and_istio
  apply_gateway_manifest
  print_apply_complete
}

main "$@"
