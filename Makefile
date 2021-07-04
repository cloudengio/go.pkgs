.PHONY: build test pullrequest

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

pr:
	for pkg in $(SUBMODULES); do \
		cd $$pkg; \
		go run cloudeng.io/go/cmd/gousage --overwrite ./...; \
		go run cloudeng.io/go/cmd/goannotate --config=../copyright-annotation.yaml --annotation=cloudeng-copyright ./...; \
		go run cloudeng.io/go/cmd/gomarkdown --overwrite --circleci=cloudengio/go.gotools --goreportcard ./...; \
		golangci-lint run ./...; \
		$(RM) go.sum; \
		go mod tidy; \
		cd ..; \
	done

