# language: zh-CN
@e2e
@smoke
@sonar-service-case
@sonar-scan-revision
@sonarqube-feature
功能: 支持扫描不同的 Git Revision

    @sonar-scan-revision-branch
    场景: 分支扫描
        假定 执行 "sonar 扫描" 脚本成功
            | command                                                                                                        |
            | bash scripts/scan.sh repos/go-example sonar-scanner -Dsonar.projectKey=revision-branch -Dsonar.branch.name=main -Dsonar.host.url=<config.{{.sonar.url}}> -Dsonar.login=<config.{{.sonar.token}}> |
            | bash scripts/scan.sh repos/go-example sonar-scanner -Dsonar.projectKey=revision-branch -Dsonar.branch.name=abc -Dsonar.host.url=<config.{{.sonar.url}}> -Dsonar.login=<config.{{.sonar.token}}> |
        并且 SonarQube 分析通过
            """
            host: <config.{{.sonar.url}}>
            token: <config.{{.sonar.token}}>
            component: revision-branch
            branch: abc
            """
        当 发送 "获取扫描结果" 请求
            """
            GET <config.{{.sonar.url}}>/api/measures/component?component=revision-branch&branch=abc&metricKeys=ncloc,coverage HTTP/1.1
            Authorization: Basic <config.{{ printf "%s:" .sonar.token | b64enc}}>
            """
        那么 HTTP 响应状态码为 "200"

    @sonar-scan-revision-pr
    场景: PR 扫描
        假定 执行 "main 分支扫描" 脚本成功
            | command                                                                                                     |
            | bash scripts/scan.sh repos/go-example sonar-scanner -Dsonar.projectKey=revision-pr -Dsonar.branch.name=main -Dsonar.host.url=<config.{{.sonar.url}}> -Dsonar.login=<config.{{.sonar.token}}> |
        并且 SonarQube 分析通过
            """
            host: <config.{{.sonar.url}}>
            token: <config.{{.sonar.token}}>
            component: revision-pr
            branch: main
            """
        并且 执行 "PR 源分支扫描" 脚本成功
            | command                                                                                                                                                                |
            | bash -c 'cd repos/go-example && sonar-scanner -Dsonar.projectKey=revision-pr -Dsonar.pullrequest.key=123 -Dsonar.pullrequest.branch=abc -Dsonar.pullrequest.base=main -Dsonar.host.url=<config.{{.sonar.url}}> -Dsonar.login=<config.{{.sonar.token}}>' |
        并且 SonarQube 分析通过
            """
            host: <config.{{.sonar.url}}>
            token: <config.{{.sonar.token}}>
            component: revision-pr
            branch: abc
            """
        当 发送 "获取扫描结果" 请求
            """
            GET <config.{{.sonar.url}}>/api/measures/component?component=revision-pr&pullRequest=123&metricKeys=ncloc,coverage HTTP/1.1
            Authorization: Basic <config.{{ printf "%s:" .sonar.token | b64enc}}>
            """
        那么 HTTP 响应状态码为 "200"
