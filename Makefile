SHELL              := /bin/bash
# go options
GO                 ?= go
LDFLAGS            :=
GOFLAGS            :=
BINDIR             ?= $(CURDIR)/bin

.PHONY: resource-auditing
resource-auditing:
	@mkdir -p $(BINDIR)
	GOOS=linux $(GO) build -o $(BINDIR) $(GOFLAGS) -ldflags '$(LDFLAGS)' antrea.io/resource-auditing

.PHONY: test
test:
	@echo "==> Running all tests <=="
	GOOS=linux $(GO) test ./test

.PHONY: audit-controller
audit-controller:
	docker build -t audit/controller -f build/images/audit-controller/Dockerfile .

.PHONY: audit-webui
audit-webui:
	docker build -t audit/webui -f build/images/webui/Dockerfile .

