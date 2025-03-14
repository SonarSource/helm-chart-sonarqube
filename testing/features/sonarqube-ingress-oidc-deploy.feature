# language: zh-CN
@sonarqube-operator-deploy
@e2e
@sonarqube-sso
功能: 支持通过 operator 部署 SonarQube

    @automated
    @priority-high
    @sonarqube-operator-deploy-sso
    场景: 通过默认配置部署 Sonarqube
        并且 集群已存在存储类
        并且 命名空间 "testing-sonarqube-sso-<template.{{randAlphaNum 4 | toLower}}>" 已存在
        并且 已导入 "SonarQube 数据库" 资源: "./testdata/resources/pg-postgresql.yaml"
        并且 已导入 "初始化 SonarQube 数据的 job" 资源: "./testdata/resources/job-init-sonar-db.yaml"
        并且 已导入 "域名 TLS 证书" 资源: "./testdata/resources/secret-tls-cert.yaml"
        并且 已导入 "自定义 root 密码" 资源: "./testdata/resources/custom-root-password.yaml"
        并且 已导入 "自定义 postgres 密码" 资源: "./testdata/resources/custom-pg-password.yaml"
        并且 已导入 "pvc" 资源: "./testdata/resources/sonarqube-pvc.yaml"
        并且 执行 "sso 配置" 脚本成功
            | command                                                                                                             |
            | sh ./testdata/script/prepare-sso-config.sh '<config.{{.acp.baseUrl}}>' '<config.{{.acp.token}}>' '<config.{{.acp.cluster}}>'  testing-sonarqube-operator http://<node.ip.first>:<nodeport.http> |
            | mkdir -p output/images                                                                                                   |
        当 已导入 "sonarqube 实例" 资源
            """
            yaml: "./testdata/ingress-oidc.yaml"
            """
        那么 "sonarqube" 可以正常访问
            """
            url: http://<node.ip.first>:<nodeport.http>
            timeout: 20m
            """
        并且 "Sonarqube 组件" 资源检查通过
            | kind        | apiVersion | name                     | path            | value | interval | timeout |
            | Deployment  | apps/v1    | ingress-oidc-sonarqube   | $.spec.replicas | 1     | 30s      | 10m     |
        并且 "ingress-oidc" 实例资源检查通过
        并且 SSO 测试通过
            """
            sonarURL: http://<node.ip.first>:<nodeport.http>/sessions/new?return_to=/projects
            acpURL: <config.{{.acp.baseUrl}}>
            acpUser: <config.{{.acp.username}}>
            acpPassword: <config.{{.acp.password}}>
            timeout: 10m
            headless: true
            """
