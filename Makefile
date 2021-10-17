.PHONY: build test pullrequest

SUBMODULES = $(wildcard */)

build:
	for pkg in $(SUBMODULES); do \
		cd $$pkg; \
		go build ./...; \
		cd ..; \
	done

prep-ci:
	mkdir -p webapp/cmd/webapp/webapp-sample/build
	mkdir -p webapp/cmd/webapp/webapp-sample/build/static/css
	mkdir -p webapp/cmd/webapp/webapp-sample/build/static/js
	mkdir -p webapp/cmd/webapp/webapp-sample/build/static/media
	touch webapp/cmd/webapp/webapp-sample/build/dummy-for-embed
	touch webapp/cmd/webapp/webapp-sample/build/static/css/dummy-for-embed
	touch webapp/cmd/webapp/webapp-sample/build/static/js/dummy-for-embed
	touch webapp/cmd/webapp/webapp-sample/build/static/media/dummy-for-embed

test-ci: prep-ci
	for pkg in $(SUBMODULES); do \
		cd $$pkg; \
		go test -failfast --covermode=atomic -race ./...; \
		cd ..; \
	done

test:
	for pkg in $(SUBMODULES); do \
		cd $$pkg; \
		go test -v ./...; \
		cd ..; \
	done

prep-lint-ci: prep-ci
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin
	go get github.com/matthewloring/validjson/cmd/validjson@latest
	go install -x github.com/matthewloring/validjson/cmd/validjson@latest

lint-ci:
	for pkg in $(SUBMODULES); do \
		cd $$pkg; \
		golangci-lint run ./...; \
		validjson ./...; \
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
		go get cloudeng.io/go/cmd/goannotate cloudeng.io/go/cmd/gousage cloudeng.io/go/cmd/gomarkdown; \
		go run cloudeng.io/go/cmd/gousage --overwrite ./...; \
		go run cloudeng.io/go/cmd/goannotate --config=../copyright-annotation.yaml --annotation=cloudeng-copyright ./...; \
		go run cloudeng.io/go/cmd/gomarkdown --overwrite --circleci=cloudengio/go.gotools --goreportcard ./...; \
		golangci-lint run ./...; \
		$(RM) go.sum; \
		go mod tidy; \
		cd ..; \
	done

