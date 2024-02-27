# This subfolder host the google cloud marketplace k8s app definition and documentation

## How to build and test the app.

### Prerequisites

please follow this [documentation](https://github.com/GoogleCloudPlatform/marketplace-k8s-app-tools/blob/master/docs/tool-prerequisites.md)

### Build and run the Deployer

in order to build the deployer, one must be at the top level of this repository and run this command.

```shell
# make sure you are on a staging account
export REGISTRY=gcr.io/$(gcloud config get-value project | tr ':' '/')
export APP_NAME=sonarqube-dce
export VERSION=10.4.0
export MINOR_VERSION=$(echo $VERSION | cut -d. -f1,2)
# Deployer does not care about patch version. see [here](https://github.com/GoogleCloudPlatform/marketplace-k8s-app-tools/blob/master/docs/building-deployer-helm.md#images-in-staging-gcr)
docker build -f google-cloud-marketplace-k8s-app/Dockerfile --tag $REGISTRY/$APP_NAME/deployer:$MINOR_VERSION .
docker push $REGISTRY/$APP_NAME/deployer:$MINOR_VERSION
```

With the deployer being built, one can deploy the app as is, with 

```shell
# make sure the namespace has been created already.
export JWT_SECRET=$(echo -n "your_secret" | openssl dgst -sha256 -hmac "your_key" -binary | base64)

mpdev install \
  --deployer=$REGISTRY/$APP_NAME/deployer:$MINOR_VERSION \
  --parameters='{"name": "sonarqube-dce-gcapp-test", "namespace": "test-ns","ApplicationNodes.jwtSecret": "dZ0EB0KxnF++nr5+4vfTCaun/eWbv6gOoXodiAMqcFo=", "postgresql.enabled": true, "jdbcOverwrite.enable": false }'
```

### App verify

On top of installing trought the deployer, we need to verify that our app comply with the requirements.

```shell
mpdev verify \
  --deployer=$REGISTRY/$APP_NAME/deployer:$MINOR_VERSION \
  --wait_timeout=1200 \
  --parameters='{"name": "sonarqube-dce-gcapp-test", "namespace": "test-ns","ApplicationNodes.jwtSecret": "dZ0EB0KxnF++nr5+4vfTCaun/eWbv6gOoXodiAMqcFo=", "postgresql.enabled": true, "jdbcOverwrite.enable": false }'
```