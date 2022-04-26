.PHONY: build test pr

SUBMODULES = $(wildcard */)

build:
	for pkg in $(SUBMODULES); do \
		cd $$pkg; \
		go build ./...; \
		cd ..; \
	done

test:
	for pkg in $(SUBMODULES); do \
		cd $$pkg; \
		go test -v ./...; \
		cd ..; \
	done

lint:
	for pkg in $(SUBMODULES); do \
		cd $$pkg; \
		golangci-lint run ./...; \
		cd ..; \
	done

deps:
	for pkg in $(SUBMODULES); do \
		cd $$pkg; \
		go get -u cloudeng.io/...; \
		go mod tidy; \
		cd ..; \
	done

pr:
	go install cloudeng.io/go/cmd/goannotate@latest cloudeng.io/go/cmd/gousage@latest cloudeng.io/go/cmd/gomarkdown@latest
	go install golang.org/x/tools/cmd/goimports@latest
	for pkg in $(SUBMODULES); do \
		cd $$pkg; \
		go run cloudeng.io/go/cmd/gousage@latest --overwrite ./...; \
		PATH=$$PATH:$$(go env GOPATH)/bin go run cloudeng.io/go/cmd/goannotate@latest --config=../copyright-annotation.yaml --annotation=cloudeng-copyright ./...; \
		go run cloudeng.io/go/cmd/gomarkdown@latest --overwrite --circleci=cloudengio/go.gotools --goreportcard ./...; \
		golangci-lint run ./...; \
		$(RM) go.sum; \
		go mod tidy; \
		cd ..; \
	done
