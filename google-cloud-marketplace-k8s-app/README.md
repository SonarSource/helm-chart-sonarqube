# This subfolder hosts the Google Cloud marketplace k8s app definition and documentation

## How to deploy SonarQube DCE on GKE

For production use cases, please refer to our official documentation
on how to [deploy a SonarQube Cluster on Kubernetes](https://docs.sonarsource.com/sonarqube/latest/setup-and-upgrade/deploy-on-kubernetes/cluster/).

## How to build and test the app

### Prerequisites

please follow this [documentation](https://github.com/GoogleCloudPlatform/marketplace-k8s-app-tools/blob/master/docs/tool-prerequisites.md)

### Build and run the Deployer

In order to build the deployer, one must be at the top level of this repository
and run this command.

```shell
# make sure you are on a staging account
export REGISTRY=gcr.io/$(gcloud config get-value project | tr ':' '/')
export APP_NAME=sonarqube-dce
export TAG=10.6.0
export MINOR_VERSION=$(echo $TAG | cut -d. -f1,2)
# Deployer does not care about patch version. see [here](https://github.com/GoogleCloudPlatform/marketplace-k8s-app-tools/blob/master/docs/building-deployer-helm.md#images-in-staging-gcr)
docker build -f google-cloud-marketplace-k8s-app/Dockerfile --build-arg REGISTRY="${REGISTRY}" --build-arg TAG="${TAG}" --tag $REGISTRY/$APP_NAME/deployer:$MINOR_VERSION .
docker push "${REGISTRY}/${APP_NAME}/deployer:${MINOR_VERSION}"
```

With the deployer being built, one can deploy the app as is, with

```shell
# make sure the namespace has been created already.
mpdev install \
  --deployer="${REGISTRY}/${APP_NAME}/deployer:${MINOR_VERSION}" \
  --parameters='{"name": "sonarqube-dce-gcapp-test", "namespace": "test-ns","ApplicationNodes.jwtSecret": "dZ0EB0KxnF++nr5+4vfTCaun/eWbv6gOoXodiAMqcFo=", "postgresql.enabled": true, "jdbcOverwrite.enable": false }'
```

### App verify

On top of installing through the deployer, we need to verify that our app
complies with the requirements.

```shell
mpdev verify \
  --deployer="${REGISTRY}/${APP_NAME}/deployer:${MINOR_VERSION}" \
  --wait_timeout=1200 \
  --parameters='{"name": "sonarqube-dce-gcapp-test", "namespace": "test-ns","ApplicationNodes.jwtSecret": "dZ0EB0KxnF++nr5+4vfTCaun/eWbv6gOoXodiAMqcFo=", "postgresql.enabled": true, "jdbcOverwrite.enable": false }'
```
