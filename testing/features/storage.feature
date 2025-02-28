# language: zh-CN

@sonarqube-chart-deploy
@sonarqube-chart-deploy-storage
功能: 支持多种存储类型部署 sonarqube

  @automated
  @priority-high
  @sonarqube-chart-deploy-storage-sc
  场景: 使用存储类方式部署 SonarQube
    假定 集群已存在存储类
    并且 命名空间 "testing-storage-sc" 已存在
    并且 已导入 "SonarQube 数据库" 资源: "./testdata/resources/pg-postgresql.yaml"
    并且 已导入 "初始化 SonarQube 数据的 job" 资源: "./testdata/resources/job-init-sonar-db.yaml"
    并且 已导入 "SonarQube 自定义 root 密码" 资源: "./testdata/resources/custom-root-password.yaml"
    并且 已导入 "自定义 postgres 密码" 资源: "./testdata/resources/custom-pg-password.yaml"
    当 使用 helm 部署实例到 "testing-storage-sc" 命名空间
      """
      chartPath: ../charts/sonarqube
      releaseName: sonarqube-sc
      values:
      - testdata/snippets/base-values.yaml
      - testdata/snippets/tpl-values-network-nodeport.yaml
      - testdata/values-storage-sc.yaml
      """
    那么 "sonarqube" 可以正常访问
      """
      url: http://<node.first>:<nodeport.http>
      timeout: 10m
      """
    并且 Pod 资源检查通过
      | name                   | path                                                                        | value                        |
      | sonarqube-sc-sonarqube | $.spec.volumes[?(@.name == 'sonarqube')][0].persistentVolumeClaim.claimName | sonarqube-sc-sonarqube       |

  @automated
  @priority-high
  @sonarqube-chart-deploy-storage-hostpath
  场景: 使用 hostpath 方式部署 sonarqube
    假定 命名空间 "testing-storage-hostpath" 已存在
    并且 已导入 "SonarQube 数据库" 资源: "./testdata/resources/pg-postgresql.yaml"
    并且 已导入 "初始化 SonarQube 数据的 job" 资源: "./testdata/resources/job-init-sonar-db.yaml"
    并且 已导入 "SonarQube 自定义 root 密码" 资源: "./testdata/resources/custom-root-password.yaml"
    并且 已导入 "自定义 postgres 密码" 资源: "./testdata/resources/custom-pg-password.yaml"
    当 使用 helm 部署实例到 "testing-storage-hostpath" 命名空间
      """
      chartPath: ../charts/sonarqube
      releaseName: sonarqube-hostpath
      values:
      - testdata/snippets/base-values.yaml
      - testdata/snippets/tpl-values-network-nodeport.yaml
      - testdata/values-storage-hostpath.yaml
      """
    那么 "sonarqube" 可以正常访问
      """
      url: http://<node.first>:<nodeport.http>
      timeout: 10m
      """
    并且 Pod 资源检查通过
      | name                         | path            | value        |
      | sonarqube-hostpath-sonarqube | $.status.hostIP | <node.first> |

  @smoke
  @automated
  @priority-high
  @sonarqube-chart-deploy-storage-pvc
  场景: 使用指定 pvc 的方式部署 sonarqube
    假定 命名空间 "testing-storage-pvc" 已存在
    并且 已导入 "SonarQube 数据库" 资源: "./testdata/resources/pg-postgresql.yaml"
    并且 已导入 "初始化 SonarQube 数据的 job" 资源: "./testdata/resources/job-init-sonar-db.yaml"
    并且 已导入 "SonarQube 自定义 root 密码" 资源: "./testdata/resources/custom-root-password.yaml"
    并且 已导入 "自定义 postgres 密码" 资源: "./testdata/resources/custom-pg-password.yaml"
    并且 已导入 "pvc" 资源: "./testdata/resources/sonarqube-pvc.yaml"
    当 使用 helm 部署实例到 "testing-storage-pvc" 命名空间
      """
      chartPath: ../charts/sonarqube
      releaseName: sonarqube-pvc
      values:
      - testdata/snippets/base-values.yaml
      - testdata/snippets/tpl-values-network-nodeport.yaml
      - testdata/values-storage-pvc.yaml
      """
    那么 "sonarqube" 可以正常访问
      """
      url: http://<node.first>:<nodeport.http>
      timeout: 10m
      """
    并且 Pod 资源检查通过
      | name                    | path                                                                        | value         |
      | sonarqube-pvc-sonarqube | $.spec.volumes[?(@.name == 'sonarqube')][0].persistentVolumeClaim.claimName | sonarqube-pvc |
    当 发送 "修改密码" 请求
      """
      POST http://<node.first>:<nodeport.http>/api/authentication/login?login=admin&password=07Apples@07Apples@ HTTP/1.1
      """
    那么 HTTP 响应状态码为 "200"
