#!/bin/bash

set -ex

SONAR_HOST=$1
SONAR_USER=$2
SONAR_PWD=$3

url="$SONAR_HOST/api/user_tokens/generate?name=my-token-$(date +%s)"
SONAR_TOKEN=$(curl -s -X POST -u "$SONAR_USER:$SONAR_PWD" "$url" | jq -r '.token')
echo "获取到的Token是: $SONAR_TOKEN"


mkdir -p ~/.m2
cat <<EOF > ~/.m2/settings.xml
<settings>
    <pluginGroups>
        <pluginGroup>org.sonarsource.scanner.maven</pluginGroup>
    </pluginGroups>
    <profiles>
        <profile>
            <id>sonar</id>
            <activation>
                <activeByDefault>true</activeByDefault>
            </activation>
            <properties>
                <!-- Optional URL to server. Default value is http://localhost:9000 -->
                <sonar.host.url>
                    $SONAR_HOST
                </sonar.host.url>
                <sonar.login>
                    $SONAR_TOKEN
                </sonar.login>
            </properties>
        </profile>
    </profiles>
</settings>
EOF


cat <<EOF > sonarqube-config.yaml
sonar:
    url: $SONAR_HOST
    token: $SONAR_TOKEN
EOF

cat ./sonarqube-config.yaml


export SONAR_HOST=$SONAR_HOST
export SONAR_TOKEN=$SONAR_TOKEN
export TESTING_CONFIG=./sonarqube-config.yaml

make sonarqube
