.PHONY: build test pr

SUBMODULES = $(wildcard */)

build:
	multimod --config=.multimod.yaml build

test:
	multimod --config=.multimod.yaml test

lint:
	multimod --config=.multimod.yaml test

deps:
	for pkg in $(SUBMODULES); do \
		cd $$pkg; \
		go get -u cloudeng.io/...; \
		go mod tidy; \
		cd ..; \
	done

pr:
	go install cloudeng.io/go/cmd/goannotate@latest \
		cloudeng.io/go/cmd/gousage@latest \
		cloudeng.io/go/cmd/gomarkdown@latest
	go install golang.org/x/tools/cmd/goimports@latest
	multimod --config=.multimod.yaml usage annotate markdown
