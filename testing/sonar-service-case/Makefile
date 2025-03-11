REPORT ?= pretty
GODOG_ARGS ?= --godog.concurrency=2 --godog.format=$(REPORT)
E2E_CONFIG ?= ./config.yaml
TEST_COMMAND ?= go test -timeout=1h -v -count 1 .
BIN ?= ./bin

TAGS ?= sonar-scan-method
test:
	$(TEST_COMMAND) $(GODOG_ARGS) --godog.tags=$(TAGS)

test-smoke:
	$(TEST_COMMAND) $(GODOG_ARGS) --godog.tags="@smoke"

test-all:
	$(TEST_COMMAND) $(GODOG_ARGS)

report:
	allure generate --clean
	allure open

help:
	go test -v . --godog.help
