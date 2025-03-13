# language: zh-CN
@e2e
@smoke
@sonar-service-case
@sonar-scan-language
功能: 支持扫描不同语言项目

    @sonar-scan-language-java
    场景: 扫描 Java 项目
        假定 执行 "sonar 扫描" 脚本成功
            | command                                                                                                                           |
            | bash scripts/scan.sh repos/maven-simple mvn verify sonar:sonar -Dsonar.projectKey=language-java -Dsonar.projectName=language-java -Dsonar.host.url=<config.{{.sonar.url}}> -Dsonar.login=<config.{{.sonar.token}}> |
            | bash scripts/wait-sonar-analysis.sh '<config.{{.sonar.url}}>' '<config.{{.sonar.token}}>' 'language-java'                         |
        当 发送 "获取扫描结果" 请求
            """
            GET <config.{{.sonar.url}}>/api/measures/component?component=language-java&branch=main&metricKeys=ncloc,coverage HTTP/1.1
            Authorization: Basic <config.{{ printf "%s:" .sonar.token | b64enc}}>
            """
        那么 HTTP 响应状态码为 "200"

    @sonar-scan-language-other
    场景大纲: 扫描其他常用语言项目
        假定 执行 "sonar 扫描" 脚本成功
            | command                                                                                                |
            | bash scripts/scan.sh <path> sonar-scanner -Dsonar.projectKey=<projectKey> -Dsonar.host.url=<config.{{.sonar.url}}> -Dsonar.login=<config.{{.sonar.token}}> |
            | bash scripts/wait-sonar-analysis.sh '<config.{{.sonar.url}}>' '<config.{{.sonar.token}}>' <projectKey> |
        当 发送 "获取扫描结果" 请求
            """
            GET <config.{{.sonar.url}}>/api/measures/component?component=<projectKey>&branch=main&metricKeys=ncloc,coverage HTTP/1.1
            Authorization: Basic <config.{{ printf "%s:" .sonar.token | b64enc}}>
            """
        那么 HTTP 响应状态码为 "200"

        例子:
            | path                 | projectKey      |
            | repos/go-example     | language-go     |
            | repos/python-example | language-python |
            | repos/nodejs-example | language-nodejs |
