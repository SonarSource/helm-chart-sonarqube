# Execute tests

## Dynamic compatibility tests

In order to execute the unit schema validation tests, you need to have golang installed according to [`../../.tool-versions`](../../.tool-versions). You can use [`mise`](https://github.com/jdx/mise) or [`asdf`](https://asdf-vm.com) to install the correct version of golang.
When the pre-requisites are fulfilled, just run:

```bash
go test -v schema_test.go
```
