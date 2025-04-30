REPORT ?= pretty
GODOG_ARGS ?= --godog.concurrency=2 --godog.format=$(REPORT)
TESTING_CONFIG ?= ./config.yaml

TEST_COMMAND ?= go test -timeout=1h -v -count 1 .
ARCH ?= amd64
BIN ?= ./bin

build:
	GOOS=linux GOARCH=$(ARCH) go test -c -o "$(BIN)/sonarqube.test" ./

generate:
	@bash hack/generate-values-snippet.sh

create-testing-ns:
	kubectl create ns bdd-testing || true

TAGS ?= @sonarqube-chart-deploy
test: create-testing-ns
	$(TEST_COMMAND) $(GODOG_ARGS) --godog.tags=$(TAGS)

test-e2e: create-testing-ns
	$(TEST_COMMAND) $(GODOG_ARGS) --godog.tags="@sonarqube-e2e"

# This test will run in other tests, so we need to add '--bdd.cleanup=false' to avoid the resources being cleanup
sonarqube: create-testing-ns
	$(TEST_COMMAND) $(GODOG_ARGS) --godog.tags="@sonarqube-feature" --bdd.cleanup=false

report:
	allure generate --clean
	allure open

help:
	go test -v . --godog.help
