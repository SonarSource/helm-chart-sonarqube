# language: zh-CN
@e2e
@smoke
@sonar-service-case
@sonar-scan-method
@sonarqube-feature
功能: 支持不同方式执行扫描
    作为开发人员
    我可以根据不同编程语言的特性选择不同的扫描方式
    以便于通过最简单的方式完成代码质量的分析

    @sonar-scan-method-cli
    场景: 通过 sonar-scanner 执行扫描
        假定 执行 "sonar 扫描" 脚本成功
            | command                                                                                              |
            | bash scripts/scan.sh repos/go-example sonar-scanner -Dsonar.projectKey=method-cli -Dsonar.host.url='<config.{{.sonar.url}}>' -Dsonar.login='<config.{{.sonar.token}}>' |
        并且 SonarQube 分析通过
            """
            host: <config.{{.sonar.url}}>
            token: <config.{{.sonar.token}}>
            component: method-cli
            """
        当 发送 "获取扫描结果" 请求
            """
            GET <config.{{.sonar.url}}>/api/measures/component?component=method-cli&branch=main&metricKeys=ncloc,coverage HTTP/1.1
            Authorization: Basic <config.{{ printf "%s:" .sonar.token | b64enc}}>
            """
        那么 HTTP 响应状态码为 "200"
        并且 HTTP 响应应包含以下 JSON 数据
            | path                                                     | value |
            | $.component.measures[?(@.metric == 'ncloc')][0].value    | 32    |
            | $.component.measures[?(@.metric == 'coverage')][0].value | 0.0  |
        并且 执行 "清理项目" 脚本成功
            | command                                                                                          |
            | bash scripts/cleanup-project.sh '<config.{{.sonar.url}}>' '<config.{{.sonar.token}}>' method-cli |

    @sonar-scan-method-maven
    场景: 通过 Maven 插件执行扫描
        假定 执行 "maven 扫描" 脚本成功
            | command                                                                                                                         |
            | bash scripts/scan.sh repos/maven-simple mvn verify sonar:sonar -Dsonar.projectKey=method-maven -Dsonar.projectName=method-maven |
        并且 SonarQube 分析通过
            """
            host: <config.{{.sonar.url}}>
            token: <config.{{.sonar.token}}>
            component: method-maven
            """
        当 发送 "获取扫描结果" 请求
            """
            GET <config.{{.sonar.url}}>/api/measures/component?component=method-maven&branch=main&metricKeys=ncloc,coverage HTTP/1.1
            Authorization: Basic <config.{{ printf "%s:" .sonar.token | b64enc}}>
            """
        那么 HTTP 响应状态码为 "200"
        并且 HTTP 响应应包含以下 JSON 数据
            | path                                                     | value |
            | $.component.measures[?(@.metric == 'ncloc')][0].value    | 92    |
            | $.component.measures[?(@.metric == 'coverage')][0].value | 50.0  |
