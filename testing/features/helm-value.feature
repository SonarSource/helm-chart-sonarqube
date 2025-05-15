# language: zh-CN
@sonarqube-chart-deploy
@sonarqube-chart-deploy-helm-value
功能: 自定义 helmValue 部署 sonarqube
  @automated
  @priority-high
  @allure.label.case_id:sonarqube-chart-deploy-no-default-plugins
  场景: 不使用默认插件安装, 开启强制认证
    假定 命名空间 "testing-sonarqube-no-default-plugins-<template.{{randAlphaNum 4 | toLower}}>" 已存在
    并且 已导入 "SonarQube 数据库" 资源: "./testdata/resources/pg-postgresql.yaml"
    并且 已导入 "初始化 SonarQube 数据的 job" 资源: "./testdata/resources/job-init-sonar-db.yaml"
    并且 已导入 "SonarQube 自定义 root 密码" 资源: "./testdata/resources/custom-root-password.yaml"
    并且 已导入 "自定义 postgres 密码" 资源: "./testdata/resources/custom-pg-password.yaml"
    当 使用 helm 部署实例到 "testing-sonarqube-no-default-plugins-<template.{{randAlphaNum 4 | toLower}}>" 命名空间
      """
      chartPath: ../charts/sonarqube
      releaseName: vaule-plugins
      timeout: 30m
      values:
      - testdata/snippets/base-values.yaml
      - testdata/snippets/tpl-values-storage-sc.yaml
      - testdata/snippets/tpl-values-network-nodeport.yaml
      - testdata/values-no-default-plugins.yaml
      """
    那么 "sonarqube" 可以正常访问
      """
      url: http://<node.ip.random.readable>:<nodeport.http>
      timeout: 30m
      """
