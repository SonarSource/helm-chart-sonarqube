# SonarQube

SonarQube 是一个开源的持续检查工具，用于自动化代码审查，帮助开发团队识别和修复代码中的缺陷、漏洞和代码异味。它支持多种编程语言，并集成到现有的开发流程中，以提升代码质量和维护性。

## 主要功能

- 代码质量分析：自动扫描代码，检测潜在的错误、漏洞和不符合编码规范的地方。
- 多语言支持：支持包括 Java、C#、JavaScript、Python、C++ 等多种编程语言。
- 持续集成集成：可以与 Jenkins、GitLab CI/CD 等持续集成工具无缝集成，实现持续代码质量监控。
- 可视化报告：提供详细的仪表板和报告，帮助团队了解代码质量状况和改进方向。
- 技术债务管理：评估和跟踪技术债务，帮助团队制定清晰的优化计划。
- 安全漏洞检测：识别代码中的安全漏洞，确保应用程序的安全性。

1. 多场景
   - pr 扫描
   - 分支扫描
2. 扫描方式
   1. cli 扫描: 不需要运行单元测试，只需要有测试报告文件和源代码即可
   2. 插件扫描（如 maven 插件）：需要运行测试，并生成测试报告文件并自动上传到 sonar server
3. 多语言支持
   - java
   - python
   - js
   - go
4. 扫描结果异步分析成功
5. 扫描结果可以通过 api 查询

```bash
# pr 扫描设置以下参数
config["sonar.pullrequest.key"] = pr.ID
config["sonar.pullrequest.branch"] = pr.Source
config["sonar.pullrequest.base"] = pr.Target
```

```bash
# 分支扫描设置以下参数
config["sonar.branch.name"] = m.getSonarBranchName()
```

## e2e 测试

官方没有直接提供 e2e 测试，但是官方提供了一些 example 项目，可以参考这些 example 项目进行 e2e 测试。

<https://github.com/SonarSource/sonar-scanning-examples>

sonar 支持配置的参数：

<https://docs.sonarsource.com/sonarqube/9.9/analyzing-source-code/analysis-parameters/>

设置 sonar server 信息及凭据

```bash
sonar-scanner -Dsonar.login=myAuthenticationToken -Dsonar.host.url=http://localhost:9000
```

### 作为 cli 扫描

需要在 `sonar-scanner.properties` 的配置文件中添加 sonar server 和 token 信息

```bash
sonar.host.url=http://192.168.137.244:32651
sonar.login=squ_0d6ccb6f5ad0bc99c6dad2e096ba1698e527a7fd
```

### 作为 maven 插件扫描

在 ~/.m2/settings.xml 中添加 sonar server 和 token 信息

```xml
<settings>
    <pluginGroups>
        <pluginGroup>org.sonarsource.scanner.maven</pluginGroup>
    </pluginGroups>
    <profiles>
        <profile>
            <id>sonar</id>
            <activation>
                <activeByDefault>true</activeByDefault>
            </activation>
            <properties>
                <!-- Optional URL to server. Default value is http://localhost:9000 -->
                <sonar.host.url>
                  http://192.168.137.244:32651
                </sonar.host.url>
                <sonar.login>
                  squ_0d6ccb6f5ad0bc99c6dad2e096ba1698e527a7fd
                </sonar.login>
            </properties>
        </profile>
     </profiles>
</settings>
```
