# Execute tests

## Dynamic compatibility tests

In order to execute the dynamic compatibility test locally, it is required to have a kind cluster (or any reachable Kubernetes context) sized for DCE (~22 GB) running (configurations stored in the default folder), Go (at least version 1.25), Helm, Kubectl.

When the pre-requisites are fulfilled, just execute the bash script `./.github/scripts/run_dynamic_compatibility_tests.sh`

```bash
./.github/scripts/run_dynamic_compatibility_tests.sh
```
