# Execute tests

## Dynamic compatibility tests

In order to execute the dynamic compatibility test locally, you need to have a k8s cluster running (configurations stored in the default folder), go (at least version 1.22).

When the pre-requisites are fulfilled, just run:

```bash
go test -v -timeout=0 ./..
```

## Schema validation tests

```bash
go test -v schema_test.go
```
