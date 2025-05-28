# SonarQube Server - Azure Marketplace Bundle

This is an empty bundle that porter has created to get you started!

To run `cpa verify`: ensure that you bind the project root, not only the `azure-marketplace-k8s-app` folder. If you do, you will get an error like this:

```
root [ /data ]# cpa verify
CPA Version:  1.3.36
By using the Azure Kubernetes CNAB Packaging Application, you agree to the License contained in the application. You can view the License at ~/LICENSE.
We collect telemetry data, if you would like to opt out of data collection please use the --telemetryOptOut flag.
Correlation Id: 64d474b1-0a27-4c48-9a33-52f3338888ae
Manifest file validated, 0 total failure(s)

Manifest verification successful.
Helm chart validated, 1 total failure(s)

  The chart path provided to the helm validate package does not exist
2025/05/20 13:08:09
Correlation Id: 64d474b1-0a27-4c48-9a33-52f3338888ae, CPABuild: 1.3.36
2025/05/20 13:08:09 The chart path provided to the helm validate package does not exist
2025/05/20 13:08:09
For more info refer to the documentation here: https://aka.ms/K8sOfferHelmChart
panic: interface conversion: interface {} is nil, not map[string]interface {}

[...]
```

This is the correct command:

```
docker run -it -v /var/run/docker.sock:/var/run/docker.sock -v $(pwd)/..:/data --entrypoint "/bin/bash" mcr.microsoft.com/container-package-app:latest
```

Before running `cpa verify` inside the container, you need to remove any dependencies:

>   There were compressed subcharts in the path: /data/charts/sonarqube
> Please decompress the chart(s) and delete the tgz file from charts folder.

Simply `rm -rf charts/sonarqube/charts` is enough to get rid of this error. But you will need to re-build the dependencies before committing anything.

Bellow is the boilerplate; kinda useful reading.

# Contents

## sonarqube-azure

We define a wrapper chart of `charts/sonarqube`. This includes those settings that are required from the Azure marketplace (e.g., `global.azure.images`), which also activate specific features in the wrapped chart.

