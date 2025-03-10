## 准备测试环境

```
python -m venv venv
source venv/bin/activate
pip install -r requirements.txt --force
```

## 生成测试报告

```
python -m pytest --cov=src --cov-report=xml:coverage.xml --junitxml=test-results.xml
```

## 执行 sonar 扫描

```
sonar-scanner -X
```
