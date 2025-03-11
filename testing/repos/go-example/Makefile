COVER_FILE := output/coverage-all.out
TEST_FILE := output/test.json
.PHONY: test
test: ## 执行 单元测试
	mkdir -p output
	go test -v -vet=off -coverpkg=./... ./... -json -coverprofile=$(COVER_FILE) -covermode=count | tee $(TEST_FILE)

scan:
	sonar-scanner -X
