SHELL              := /bin/bash
# go options
GO                 ?= go
LDFLAGS            :=
GOFLAGS            :=
BINDIR             ?= $(CURDIR)/bin

all: bin

.PHONY: bin
bin:
	@mkdir -p $(BINDIR)
	GOOS=linux $(GO) build -o $(BINDIR) $(GOFLAGS) -ldflags '$(LDFLAGS)' antrea.io/resource-auditing/cmd/...

.PHONY: test
test:
	@echo "==> Running all tests <=="
	GOOS=linux $(GO) test ./...

.PHONY: audit-controller
audit-controller:
	docker build -t audit/controller -f build/images/audit-controller/Dockerfile .

.PHONY: audit-webui
audit-webui:
	docker build -t audit/webui -f build/images/webui/Dockerfile .

# code linting
.golangci-bin:
	@echo "===> Installing Golangci-lint <==="
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $@ v1.41.1

.PHONY: golangci
golangci: .golangci-bin
	@echo "===> Running golangci <==="
	@GOOS=linux .golangci-bin/golangci-lint run -c .golangci.yml

.PHONY: golangci-fix
golangci-fix: .golangci-bin
	@echo "===> Running golangci-fix <==="
	@GOOS=linux .golangci-bin/golangci-lint run -c .golangci.yml --fix
