# Unit Compatibility test

Unit compatibility tests are used with helm template and kubeconform,

The goal is to run templating for a lot of values files and verify if they comply with the openApi spec of the targeted kubernetes version, ensuring good structure and theorical compatibility.

In order to run them you need helm and kubeconform binaries.

Then:

```bash
cd charts/sonarqube
../../.github/scripts/unit_helm_compatibility_test.sh 
```

This will run the test against the 1.25 kubernetes version, you can specify another version with the ```KUBE_VERSION``` env var in the x.y.z format.

```bash
KUBE_VERSION=1.19.6 ../../.github/scripts/unit_helm_compatibility_test.sh
```

## Fixtures

The fixtures test's goal is to make sure the behavior of our templating stays the same between changes, catching regressions, and making sure any changes are explicitly seen and reviewed.

They use a pre-commit hook to template our chart against the unit compatibility test values; then dev commits the resulting yamls. That will allow PR review to ensure we have not introduced unwanted changes.
