# language: zh-CN

@sonarqube-chart-deploy
@sonarqube-chart-deploy-storage
功能: 支持多种存储类型部署 sonarqube

  @automated
  @priority-high
  @allure.label.case_id:sonarqube-chart-deploy-storage-sc
  场景: 使用存储类方式部署 SonarQube
    假定 集群已存在存储类
    并且 命名空间 "testing-sonarqube-storage-sc-<template.{{randAlphaNum 4 | toLower}}>" 已存在
    并且 已导入 "SonarQube 数据库" 资源: "./testdata/resources/pg-postgresql.yaml"
    并且 已导入 "初始化 SonarQube 数据的 job" 资源: "./testdata/resources/job-init-sonar-db.yaml"
    并且 已导入 "SonarQube 自定义 root 密码" 资源: "./testdata/resources/custom-root-password.yaml"
    并且 已导入 "自定义 postgres 密码" 资源: "./testdata/resources/custom-pg-password.yaml"
    当 使用 helm 部署实例到 "testing-sonarqube-storage-sc-<template.{{randAlphaNum 4 | toLower}}>" 命名空间
      """
      chartPath: ../charts/sonarqube
      releaseName: sonarqube-sc
      timeout: 30m
      values:
      - testdata/snippets/base-values.yaml
      - testdata/snippets/tpl-values-network-nodeport.yaml
      - testdata/snippets/tpl-values-storage-sc.yaml
      """
    那么 "sonarqube" 可以正常访问
      """
      url: http://<node.ip.random.readable>:<nodeport.http>
      timeout: 30m
      """
    并且 Pod 资源检查通过
      | name                   | path                                                                        | value                        |
      | sonarqube-sc-sonarqube | $.spec.volumes[?(@.name == 'sonarqube')][0].persistentVolumeClaim.claimName | sonarqube-sc-sonarqube       |
    当 执行 "sonar 扫描" 脚本成功
      | command                                                                                              |`
      | bash scripts/scan_with_notoken.sh 'http://<node.ip.random.readable>:<nodeport.http>' admin 07Apples@07Apples@ repos/go-example sonar-scanner -Dsonar.host.url='http://<node.ip.random.readable>:<nodeport.http>' -Dsonar.projectKey=method-cli |
    并且 SonarQube 分析通过
      """
      host: http://<node.ip.random.readable>:<nodeport.http>
      user: admin
      pwd: 07Apples@07Apples@
      component: method-cli
      """
    并且 发送 "获取扫描结果" 请求
        """
        GET http://<node.ip.random.readable>:<nodeport.http>/api/measures/component?component=method-cli&branch=main&metricKeys=ncloc,coverage HTTP/1.1
        Authorization: Basic YWRtaW46MDdBcHBsZXNAMDdBcHBsZXNA
        """
    那么 HTTP 响应状态码为 "200"
    并且 HTTP 响应应包含以下 JSON 数据
        | path                                                     | value |
        | $.component.measures[?(@.metric == 'ncloc')][0].value    | 32    |
        | $.component.measures[?(@.metric == 'coverage')][0].value | 0.0  |

  @automated
  @priority-high
  @allure.label.case_id:sonarqube-chart-deploy-storage-hostpath
  场景: 使用 hostpath 方式部署 sonarqube
    假定 命名空间 "testing-sonarqube-storage-hostpath-<template.{{randAlphaNum 4 | toLower}}>" 已存在
    并且 已导入 "SonarQube 数据库" 资源: "./testdata/resources/pg-postgresql.yaml"
    并且 已导入 "初始化 SonarQube 数据的 job" 资源: "./testdata/resources/job-init-sonar-db.yaml"
    并且 已导入 "SonarQube 自定义 root 密码" 资源: "./testdata/resources/custom-root-password.yaml"
    并且 已导入 "自定义 postgres 密码" 资源: "./testdata/resources/custom-pg-password.yaml"
    当 使用 helm 部署实例到 "testing-sonarqube-storage-hostpath-<template.{{randAlphaNum 4 | toLower}}>" 命名空间
      """
      chartPath: ../charts/sonarqube
      releaseName: sonarqube-hostpath
      timeout: 30m
      values:
      - testdata/snippets/base-values.yaml
      - testdata/snippets/tpl-values-network-nodeport.yaml
      - testdata/values-storage-hostpath.yaml
      """
    那么 "sonarqube" 可以正常访问
      """
      url: http://<node.ip.random.readable>:<nodeport.http>
      timeout: 30m
      """
    并且 Pod 资源检查通过
      | name                         | path                | value        |
      | sonarqube-hostpath-sonarqube | $.spec.volumes[?(@.name == sonarqube)][0].hostPath.type | DirectoryOrCreate |

  @smoke
  @automated
  @priority-high
  @allure.label.case_id:sonarqube-chart-deploy-storage-pvc
  场景: 使用指定 pvc 的方式部署 sonarqube
    假定 命名空间 "testing-sonarqube-storage-pvc-<template.{{randAlphaNum 4 | toLower}}>" 已存在
    并且 集群已存在存储类
    并且 已导入 "SonarQube 数据库" 资源: "./testdata/resources/pg-postgresql.yaml"
    并且 已导入 "初始化 SonarQube 数据的 job" 资源: "./testdata/resources/job-init-sonar-db.yaml"
    并且 已导入 "SonarQube 自定义 root 密码" 资源: "./testdata/resources/custom-root-password.yaml"
    并且 已导入 "自定义 postgres 密码" 资源: "./testdata/resources/custom-pg-password.yaml"
    并且 已导入 "pvc" 资源: "./testdata/resources/sonarqube-pvc.yaml"
    当 使用 helm 部署实例到 "testing-sonarqube-storage-pvc-<template.{{randAlphaNum 4 | toLower}}>" 命名空间
      """
      chartPath: ../charts/sonarqube
      releaseName: sonarqube-pvc
      timeout: 30m
      values:
      - testdata/snippets/base-values.yaml
      - testdata/snippets/tpl-values-network-nodeport.yaml
      - testdata/values-storage-pvc.yaml
      """
    那么 "sonarqube" 可以正常访问
      """
      url: http://<node.ip.random.readable>:<nodeport.http>
      timeout: 30m
      """
    并且 Pod 资源检查通过
      | name                    | path                                                                        | value         |
      | sonarqube-pvc-sonarqube | $.spec.volumes[?(@.name == 'sonarqube')][0].persistentVolumeClaim.claimName | sonarqube-pvc |
    假定 执行 "maven 扫描" 脚本成功
      | command                                                                                                                         |
      | bash scripts/scan_with_notoken.sh http://<node.ip.random.readable>:<nodeport.http> admin 07Apples@07Apples@ repos/maven-simple mvn verify sonar:sonar -Dsonar.projectKey=method-maven -Dsonar.projectName=method-maven -Dsonar.host.url=http://<node.ip.random.readable>:<nodeport.http> |
    并且 SonarQube 分析通过
      """
      host: http://<node.ip.random.readable>:<nodeport.http>
      user: admin
      pwd: 07Apples@07Apples@
      component: method-maven
      """
    当 发送 "获取扫描结果" 请求
        """
        GET http://<node.ip.random.readable>:<nodeport.http>/api/measures/component?component=method-maven&branch=main&metricKeys=ncloc,coverage HTTP/1.1
        Authorization: Basic YWRtaW46MDdBcHBsZXNAMDdBcHBsZXNA
        """
    那么 HTTP 响应状态码为 "200"
    并且 HTTP 响应应包含以下 JSON 数据
        | path                                                     | value |
        | $.component.measures[?(@.metric == 'ncloc')][0].value    | 92    |
        | $.component.measures[?(@.metric == 'coverage')][0].value | 50.0  |
