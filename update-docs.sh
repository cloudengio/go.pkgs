#!/bin/bash
for pkg in $(find * -type d -maxdepth 0); do
	cd $pkg
	go run cloudeng.io/go/cmd/gousage --overwrite ./...
	go run cloudeng.io/go/cmd/goannotate --config=../copyright-annotation.yaml --annotation=cloudeng-copyright ./...
	go run cloudeng.io/go/cmd/gomarkdown --overwrite --circleci=cloudengio/go.gotools --goreportcard ./...
	rm go.sum
	go mod tidy
	cd ..
done
