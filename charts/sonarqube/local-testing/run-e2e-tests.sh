#!/usr/bin/env bash
# Runs the Agentic Harness reference test flows (demo/pressure/backpressure — defined in
# agentic-workflows/private/integration-harness/scripts/) against a real `kind` deployment of
# this chart with agenticHarness.enabled=true (see README.md in this folder for the setup
# sequence). The reference scripts assume a docker-compose environment (they exec into services
# via `docker compose exec`), so this wraps them with a shim that translates those calls into
# `kubectl exec` against the matching pod — the reference scripts themselves run unmodified.
#
# Usage:
#   ./run-e2e-tests.sh demo|pressure|backpressure|all
#
# Env vars:
#   NAMESPACE       - k8s namespace the chart is installed in (default: sonarqube)
#   HARNESS_DIR     - path to agentic-workflows/private/integration-harness (default: guessed
#                     relative to sonarqube-unification checked out as a sibling of this repo)
#   SQ_LOCAL_PORT   - local port for the SonarQube port-forward (default: 9000)
#   N               - concurrency for pressure/backpressure (passed through to the reference
#                      scripts' own N env var — see their own defaults if unset)
set -euo pipefail

export NAMESPACE="${NAMESPACE:-sonarqube}"
HARNESS_DIR="${HARNESS_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../../sonarqube-unification/agentic-workflows/private/integration-harness" 2>/dev/null && pwd)}"
SQ_LOCAL_PORT="${SQ_LOCAL_PORT:-9000}"

if [ -z "$HARNESS_DIR" ] || [ ! -f "$HARNESS_DIR/scripts/demo.sh" ]; then
  echo "error: can't find agentic-workflows/private/integration-harness — set HARNESS_DIR explicitly" >&2
  exit 1
fi

mode="${1:-}"
case "$mode" in
  demo|pressure|backpressure|all) ;;
  *) echo "usage: $0 demo|pressure|backpressure|all" >&2; exit 1 ;;
esac

docker() {
  if [ "$1" = "compose" ] && [ "$2" = "exec" ]; then
    shift 2
    if [ "$1" = "-T" ]; then shift; fi
    local svc="$1"; shift
    local pod container
    case "$svc" in
      orchestrator) pod=$(kubectl get pod -n "$NAMESPACE" -l app=sonarqube-agentic-orchestrator -o jsonpath='{.items[0].metadata.name}'); container=agentic-orchestrator ;;
      hunter-runtime) pod=$(kubectl get pod -n "$NAMESPACE" -l app=sonarqube-agentic-runtime-hunter -o jsonpath='{.items[0].metadata.name}'); container=agentic-runtime ;;
      *) echo "run-e2e-tests: unmapped service '$svc'" >&2; return 1 ;;
    esac
    kubectl exec -i -n "$NAMESPACE" -c "$container" "$pod" -- "$@"
  else
    command docker "$@"
  fi
}
export -f docker

pf_pid=""
cleanup() { [ -n "$pf_pid" ] && kill "$pf_pid" 2>/dev/null || true; }
trap cleanup EXIT

kubectl port-forward -n "$NAMESPACE" "svc/sonarqube-sonarqube" "${SQ_LOCAL_PORT}:9000" >/tmp/run-e2e-tests-pf.log 2>&1 &
pf_pid=$!
for _ in $(seq 1 20); do
  curl -sf --max-time 1 "http://localhost:${SQ_LOCAL_PORT}/api/system/status" >/dev/null 2>&1 && break
  sleep 1
done

cd "$HARNESS_DIR"
export SQ="http://localhost:${SQ_LOCAL_PORT}"
export SONAR_TOKEN
SONAR_TOKEN=$(SQ="$SQ" bash scripts/get-token.sh)

run_demo() { echo "== demo =="; bash scripts/demo.sh; }
run_pressure() { echo "== pressure (N=${N:-8}) =="; bash scripts/pressure.sh; }
run_backpressure() { echo "== backpressure (N=${N:-6}) =="; bash scripts/backpressure.sh; }

case "$mode" in
  demo) run_demo ;;
  pressure) run_pressure ;;
  backpressure) run_backpressure ;;
  all) run_demo; echo; run_pressure; echo; run_backpressure ;;
esac
