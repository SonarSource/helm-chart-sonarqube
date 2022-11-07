## Execute tests

In order to execute the dynamic compability test locally, you need to have a k8s cluster running (configurations stored in the default folder), go (at least version 1.13).

When the pre-requisites are fullfilled, just run:

```
go test -timeout=0 -v sonarqube_standard_dynamic_test.go pod_utils.go
```