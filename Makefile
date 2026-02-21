.PHONY: build lint clean install install-lint

BINARY_NAME=kopi
GO=go
LINTER=golangci-lint
GOLANGCI_LINT_VERSION=v2.10.1

build:
	$(GO) build -o $(BINARY_NAME) .

lint:
	$(LINTER) run ./...

install-lint:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin $(GOLANGCI_LINT_VERSION)

clean:
	rm -f $(BINARY_NAME)

install: build
	cp $(BINARY_NAME) $$HOME/.local/bin/kubectl-$(BINARY_NAME)
