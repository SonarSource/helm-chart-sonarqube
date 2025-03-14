# language: zh-CN
@sonarqube-chart-deploy
@sonarqube-chart-deploy-network
功能: 支持多种网络模式部署 sonarqube

  @automated
  @priority-high
  @sonarqube-chart-deploy-network-http
  场景: 使用 http 方式部署 sonarqube
    假定 集群已安装 ingress controller
    并且 已添加域名解析
      | domain                        | ip           |
      | test-ingress-http.example.com | <ingress-ip> |
    并且 命名空间 "testing-sonarqube-http-<template.{{randAlphaNum 4 | toLower}}>" 已存在
    并且 已导入 "SonarQube 数据库" 资源: "./testdata/resources/pg-postgresql.yaml"
    并且 已导入 "初始化 SonarQube 数据的 job" 资源: "./testdata/resources/job-init-sonar-db.yaml"
    并且 已导入 "SonarQube 自定义 root 密码" 资源: "./testdata/resources/custom-root-password.yaml"
    并且 已导入 "自定义 postgres 密码" 资源: "./testdata/resources/custom-pg-password.yaml"
    当 使用 helm 部署实例到 "testing-sonarqube-http-<template.{{randAlphaNum 4 | toLower}}>" 命名空间
      """
      chartPath: ../charts/sonarqube
      releaseName: sonarqube-http
      timeout: 20m
      values:
      - testdata/snippets/base-values.yaml
      - testdata/snippets/tpl-values-storage-hostpath.yaml
      - testdata/values-network-http.yaml
      """
    那么 "sonarqube" 可以正常访问
      """
      url: http://test-ingress-http.example.com
      timeout: 20m
      """

  @smoke
  @automated
  @priority-high
  @sonarqube-chart-deploy-network-https
  场景: 使用 https 方式部署 sonarqube
    假定 集群已安装 ingress controller
    并且 已添加域名解析
      | domain                         | ip           |
      | test-ingress-https.example.com | <ingress-ip> |
    并且 命名空间 "testing-sonarqube-https-<template.{{randAlphaNum 4 | toLower}}>" 已存在
    并且 已导入 "SonarQube 数据库" 资源: "./testdata/resources/pg-postgresql.yaml"
    并且 已导入 "初始化 SonarQube 数据的 job" 资源: "./testdata/resources/job-init-sonar-db.yaml"
    并且 已导入 "SonarQube 自定义 root 密码" 资源: "./testdata/resources/custom-root-password.yaml"
    并且 已导入 "自定义 postgres 密码" 资源: "./testdata/resources/custom-pg-password.yaml"
    并且 已导入 "tls 证书" 资源: "./testdata/resources/secret-tls-cert.yaml"
    当 使用 helm 部署实例到 "testing-sonarqube-https-<template.{{randAlphaNum 4 | toLower}}>" 命名空间
      """
      chartPath: ../charts/sonarqube
      releaseName: sonarqube-https
      timeout: 20m
      values:
      - testdata/snippets/base-values.yaml
      - testdata/snippets/tpl-values-storage-hostpath.yaml
      - testdata/values-network-https.yaml
      """
    那么 "sonarqube" 可以正常访问
      """
      url: https://test-ingress-https.example.com
      timeout: 20m
      """
    当 执行 "sonar 扫描" 脚本成功
      | command                                                                                                                           |
      | bash -x testdata/script/wait-sonar-analysis.sh 'https://test-ingress-https.example.com' '07Apples@07Apples@' 'language-java'  -Dsonar.login='admin' -Dsonar.password='07Apples@07Apples@'|
    并且 发送 "获取扫描结果" 请求
      """
      GET https://test-ingress-https.example.com/api/measures/component?component=language-java&branch=main&metricKeys=ncloc,coverage HTTP/1.1
      Authorization: Basic YWRtaW46MDdBcHBsZXNAMDdBcHBsZXNA
      """
    那么 HTTP 响应状态码为 "200"
   
  @smoke
  @automated
  @priority-high
  @sonarqube-chart-deploy-network-nodeport
  场景: 使用 nodeport 方式部署 sonarqube
    假定 命名空间 "testing-sonarqube-nodeport-<template.{{randAlphaNum 4 | toLower}}>" 已存在
    并且 已导入 "SonarQube 数据库" 资源: "./testdata/resources/pg-postgresql.yaml"
    并且 已导入 "初始化 SonarQube 数据的 job" 资源: "./testdata/resources/job-init-sonar-db.yaml"
    并且 已导入 "SonarQube 自定义 root 密码" 资源: "./testdata/resources/custom-root-password.yaml"
    并且 已导入 "自定义 postgres 密码" 资源: "./testdata/resources/custom-pg-password.yaml"
    当 使用 helm 部署实例到 "testing-sonarqube-nodeport-<template.{{randAlphaNum 4 | toLower}}>" 命名空间
      """
      chartPath: ../charts/sonarqube
      releaseName: sonarqube-nodeport
      timeout: 20m
      values:
      - testdata/snippets/base-values.yaml
      - testdata/snippets/tpl-values-storage-hostpath.yaml
      - testdata/values-network-nodeport.yaml
      """
    那么 "sonarqube" 可以正常访问
      """
      url: http://<node.ip.first>:<nodeport.http>
      timeout: 20m
      """
    当 执行 "sonar 扫描" 脚本成功
      | command                                                                                                |
      | bash scripts/scan.sh <path> sonar-scanner -Dsonar.projectKey=<projectKey> -Dsonar.host.url='http://<node.ip.first>:<nodeport.http>' -Dsonar.login='admin' -Dsonar.password='07Apples@07Apples@' |
      | bash scripts/wait-sonar-analysis.sh 'http://<node.ip.first>:<nodeport.http>' '07Apples@07Apples@' <projectKey> |
    并且 发送 "获取扫描结果" 请求
      """
      GET http://<node.ip.first>:<nodeport.http>/api/measures/component?component=<projectKey>&branch=main&metricKeys=ncloc,coverage HTTP/1.1
      Authorization: Basic YWRtaW46MDdBcHBsZXNAMDdBcHBsZXNA
      """
    那么 HTTP 响应状态码为 "200"

      例子:
        | path                 | projectKey      |
        | repos/go-example     | language-go     |

