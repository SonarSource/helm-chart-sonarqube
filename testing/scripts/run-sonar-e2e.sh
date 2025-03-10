set -ex

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
source "$SCRIPT_DIR/get-token.sh"

SONAR_HOST=$1
SONAR_PWD=$2

SONAR_TOKEN=$(get_token "$SONAR_HOST" "$SONAR_PWD")
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

cd sonar-service-case

cat <<EOF > config.yaml
sonar:
    url: $SONAR_HOST
    token: $SONAR_TOKEN
EOF

export SONAR_HOST=$SONAR_HOST
export SONAR_TOKEN=$SONAR_TOKEN

/tools/bin/sonar-service-case.test --godog.concurrency=1 --godog.tags='@sonar-service-case'
