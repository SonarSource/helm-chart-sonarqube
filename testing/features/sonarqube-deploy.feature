# language: zh-CN
@sonarqube-operator-deploy
@e2e
@sonarqube-e2e
功能: 支持通过 operator 部署 SonarQube

    @automated
    @priority-high
    @sonarqube-operator-deploy
    场景: 通过默认配置部署 Sonarqube
        假定 集群已存在默认存储类
        并且 命名空间 "testing-sonarqube-operator-<template.{{randAlphaNum 4 | toLower}}>" 已存在
        并且 已导入 "SonarQube 数据库" 资源: "./testdata/resources/pg-postgresql.yaml"
        并且 已导入 "初始化 SonarQube 数据的 job" 资源: "./testdata/resources/job-init-sonar-db.yaml"
        并且 已导入 "自定义 root 密码" 资源: "./testdata/resources/custom-root-password.yaml"
        并且 已导入 "自定义 postgres 密码" 资源: "./testdata/resources/custom-pg-password.yaml"
        并且 已导入 "pvc" 资源: "./testdata/resources/sonarqube-pvc.yaml"
        当 已导入 "sonarqube 实例" 资源
            """
            yaml: "./testdata/sonarqube.yaml"
            """
        那么 "sonarqube" 可以正常访问
            """
            url: http://<node.ip.first>:<nodeport.http>
            timeout: 10m
            """
        并且 "Sonarqube 组件" 资源检查通过
            | kind        | apiVersion | name                     | path            | value | interval | timeout |
            | Deployment  | apps/v1    | sonarqube-test-sonarqube   | $.spec.replicas | 1     | 30s      | 10m     |
        并且 "sonarqube-test" 实例资源检查通过
        并且 执行 "Sonarqube 官方 e2e" 脚本成功
          | command                                                                                                                                                 |
          | bash scripts/run-sonar-e2e.sh http://<node.ip.first>:<nodeport.http> admin 07Apples@07Apples@ |