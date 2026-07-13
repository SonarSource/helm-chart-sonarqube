# Local `kind` testing for the Agentic Harness

Exercises `agenticHarness.enabled=true` end to end against a real cluster, without needing any
published images or an external Postgres/S3.

## 0. Create the cluster with Calico (required for real NetworkPolicy enforcement)

Stock `kind` (and this repo's custom `kindest/node:v1.35.0-sonar` image) ships `kindnet`, which
doesn't enforce `ipBlock`/CIDR peers at all — every egress rule this feature needs for external
endpoints (LLM, and any real external DB/storage) is CIDR-based, so `agenticHarness.networkPolicy`
can't be meaningfully verified without swapping in a CNI that actually implements `ipBlock`:

```bash
cat > /tmp/kind-config-calico.yaml << 'EOF'
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
networking:
  disableDefaultCNI: true
  podSubnet: "192.168.0.0/16"
nodes:
  - role: control-plane
  - role: worker
  - role: worker
  - role: worker
  - role: worker
  - role: worker
EOF
kind create cluster --name my-cluster --image kindest/node:v1.35.0-sonar --config /tmp/kind-config-calico.yaml

kubectl apply -f https://raw.githubusercontent.com/projectcalico/calico/v3.29.1/manifests/calico.yaml
kubectl wait --for=condition=ready pod -l k8s-app=calico-node -n kube-system --timeout=180s
```

(adjust worker count / node image to match whatever cluster you're actually replicating)

## 1. Provision Postgres and MinIO in-cluster

```bash
kubectl create namespace sonarqube --dry-run=client -o yaml | kubectl apply -f -

NAME=external-postgres NAMESPACE=sonarqube \
  VALUES_FILE=charts/sonarqube/local-testing/postgres-values.yaml \
  ./.github/scripts/setup_external_postgres.sh

NAMESPACE=sonarqube ./.github/scripts/setup_external_minio.sh
```

Confirm the actual Service names/ports match what's in `agentic-harness-kind-values.yaml`
(Bitnami chart defaults, derived from the release names above — verify, don't assume):

```bash
kubectl get svc -n sonarqube
```

## 2. Build and load the orchestrator + hunter-runtime images

```bash
cd ../sonarqube-unification/agentic-workflows
./gradlew :private:agent-orchestrator:agent-orchestrator-app:bootJar

docker build -t agentic-orchestrator:local private/agent-orchestrator/agent-orchestrator-app
docker build -t agentic-hunter-runtime:local private/integration-harness/mocks/runtime

kind load docker-image agentic-orchestrator:local --name <your-kind-cluster-name>
kind load docker-image agentic-hunter-runtime:local --name <your-kind-cluster-name>
```

## 3. Install the chart

```bash
helm upgrade -i -n sonarqube sonarqube ../../.. \
  -f agentic-harness-kind-values.yaml
```

(path is relative to this folder — adjust if running from elsewhere)

## 4. Verify

```bash
kubectl get pods -n sonarqube
kubectl logs -n sonarqube deploy/sonarqube-sonarqube-agentic-orchestrator
```

## 5. Run the reference test flows

`run-e2e-tests.sh` runs the reference harness's own `demo.sh`/`pressure.sh`/`backpressure.sh`
(`agentic-workflows/private/integration-harness/scripts/`) against this real deployment,
completely unmodified — it wraps `docker compose exec` calls with a shim that translates them to
`kubectl exec` against the matching pod, and manages its own `kubectl port-forward` to SonarQube.

```bash
./run-e2e-tests.sh demo           # one job, full trace: create -> dispatch -> clone -> push -> LLM -> fire-back
N=8 ./run-e2e-tests.sh pressure   # N concurrent jobs through the real flow, final status tally
N=6 ./run-e2e-tests.sh backpressure  # burst against a single runtime replica: expect 1x202, (N-1)x429
./run-e2e-tests.sh all            # runs all three in sequence
```

If `agentic-workflows` isn't checked out as a sibling of `helm-chart-sonarqube` under the same
parent directory, set `HARNESS_DIR` explicitly. `NAMESPACE` (default `sonarqube`) and
`SQ_LOCAL_PORT` (default `9000`) are also overridable.

**Reaching a real LLM**: the reference runtime's `LLM_URL` (see `agentic-harness-kind-values.yaml`)
points at `https://api.anthropic.com/v1/messages`. With no API key configured this legitimately
gets an HTTP 401 back — that still counts as a working call and the job reports `SUCCEEDED`. On a
network that TLS-inspects outbound traffic (common on corporate networks), the runtime needs that
network's CA cert imported into its JRE trust store or you'll see `SSLHandshakeException`/`PKIX
path building failed` instead of a clean 401 — mount a `.crt`/`.pem` at `/custom-ca` on the runtime
container (see `mocks/runtime/entrypoint.sh`) and make sure `containerSecurityContext.runAsUser`
matches the image's actual non-root uid (the mock runtime uses uid 10001, not the chart's uid-1000
default — see the comment above `runtimes.hunter.containerSecurityContext` in `values.yaml`),
since the JRE trust store is chowned to that specific uid for writability.

**Verified with Calico installed** (step 0): the runtime really can reach only Anthropic's CIDR and
nothing else (confirmed — a request to `api.anthropic.com` gets a real HTTP response, a request to
an arbitrary other host times out), and the runtime is really reachable only from the orchestrator
(confirmed — an unrelated pod's request times out, the orchestrator's succeeds). Stock `kindnet`
doesn't enforce `ipBlock` peers at all, so none of this is verifiable without Calico.

**A NetworkPolicy `ipBlock`/CIDR rule can never target a Service's ClusterIP, on any CNI** —
kube-proxy DNATs the ClusterIP to the backing pod's real IP before policy enforcement sees the
packet, so the CIDR never matches what's actually on the wire. This bit both the orchestrator's DB
connection and its direct storage (MinIO) upload, since both are in-cluster Services in this local
test — fixed by using `egressAllow`'s `podSelector` form (see `values.yaml`'s comment on
`orchestrator.egressAllow`) instead of `cidr` for those two entries; `agentic-harness-kind-values.yaml`
already does this. This isn't a kind-specific quirk — it'd bite in production too for anyone
self-hosting their DB or S3 gateway in-cluster rather than using a genuinely external endpoint.
