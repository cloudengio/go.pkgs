.PHONY: prep-test test prep-lint lint

SUBMODULES = $(wildcard */)

prep-test:
	go run .circlci/runner.go --prep --test=false

test:
	go run .circlci/runner.go --prep=false --test

prep-lint:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin
	go get github.com/matthewloring/validjson/cmd/validjson@latest
	go install -x github.com/matthewloring/validjson/cmd/validjson@latest

lint:
	for pkg in $(SUBMODULES); do \
		cd $$pkg; \
		golangci-lint run ./...; \
		validjson ./...; \
		cd ..; \
	done
