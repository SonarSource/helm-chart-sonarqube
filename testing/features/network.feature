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
    并且 命名空间 "testing-sonarqube-http" 已存在
    并且 已导入 "SonarQube 数据库" 资源: "./testdata/resources/pg-postgresql.yaml"
    并且 已导入 "初始化 SonarQube 数据的 job" 资源: "./testdata/resources/job-init-sonar-db.yaml"
    并且 已导入 "SonarQube 自定义 root 密码" 资源: "./testdata/resources/custom-root-password.yaml"
    并且 已导入 "自定义 postgres 密码" 资源: "./testdata/resources/custom-pg-password.yaml"
    当 使用 helm 部署实例到 "testing-sonarqube-http" 命名空间
      """
      chartPath: ../charts/sonarqube
      releaseName: sonarqube-http
      values:
      - testdata/snippets/base-values.yaml
      - testdata/snippets/tpl-values-storage-hostpath.yaml
      - testdata/values-network-http.yaml
      """
    那么 "sonarqube" 可以正常访问
      """
      url: http://test-ingress-http.example.com
      timeout: 10m
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
    并且 命名空间 "testing-sonarqube-https" 已存在
    并且 已导入 "SonarQube 数据库" 资源: "./testdata/resources/pg-postgresql.yaml"
    并且 已导入 "初始化 SonarQube 数据的 job" 资源: "./testdata/resources/job-init-sonar-db.yaml"
    并且 已导入 "SonarQube 自定义 root 密码" 资源: "./testdata/resources/custom-root-password.yaml"
    并且 已导入 "自定义 postgres 密码" 资源: "./testdata/resources/custom-pg-password.yaml"
    并且 已导入 "tls 证书" 资源: "./testdata/resources/secret-tls-cert.yaml"
    当 使用 helm 部署实例到 "testing-sonarqube-https" 命名空间
      """
      chartPath: ../charts/sonarqube
      releaseName: sonarqube-https
      values:
      - testdata/snippets/base-values.yaml
      - testdata/snippets/tpl-values-storage-hostpath.yaml
      - testdata/values-network-https.yaml
      """
    那么 "sonarqube" 可以正常访问
      """
      url: https://test-ingress-https.example.com
      timeout: 10m
      """

  @smoke
  @automated
  @priority-high
  @sonarqube-chart-deploy-network-nodeport
  场景: 使用 nodeport 方式部署 sonarqube
    假定 命名空间 "testing-sonarqube-nodeport" 已存在
    并且 已导入 "SonarQube 数据库" 资源: "./testdata/resources/pg-postgresql.yaml"
    并且 已导入 "初始化 SonarQube 数据的 job" 资源: "./testdata/resources/job-init-sonar-db.yaml"
    并且 已导入 "SonarQube 自定义 root 密码" 资源: "./testdata/resources/custom-root-password.yaml"
    并且 已导入 "自定义 postgres 密码" 资源: "./testdata/resources/custom-pg-password.yaml"
    当 使用 helm 部署实例到 "testing-sonarqube-nodeport" 命名空间
      """
      chartPath: ../charts/sonarqube
      releaseName: sonarqube-nodeport
      values:
      - testdata/snippets/base-values.yaml
      - testdata/snippets/tpl-values-storage-hostpath.yaml
      - testdata/values-network-nodeport.yaml
      """
    那么 "sonarqube" 可以正常访问
      """
      url: http://<node.first>:<nodeport.http>
      timeout: 10m
      """