PLATFORM=$(shell uname -s | tr '[:upper:]' '[:lower:]')
VERSION := $(shell grep -Eo '(v[0-9]+[\.][0-9]+[\.][0-9]+(-[a-zA-Z0-9]*)?)' version.go)

.PHONY: build build-server build-examples docker release check

build: check build-server

build-server:
	CGO_ENABLED=1 go build -o ./bin/server github.com/moov-io/customers/cmd/server

check:
	go fmt ./...
	@mkdir -p ./bin/

.PHONY: admin
admin:
# Versions from https://github.com/OpenAPITools/openapi-generator/releases
	@chmod +x ./openapi-generator
	@rm -rf ./admin
	OPENAPI_GENERATOR_VERSION=4.2.0 ./openapi-generator generate --package-name admin -i openapi-admin.yaml -g go -o ./admin
	rm -f admin/go.mod admin/go.sum
	go fmt ./...
	go test ./admin

.PHONY: client
client:
# Versions from https://github.com/OpenAPITools/openapi-generator/releases
	@chmod +x ./openapi-generator
	@rm -rf ./client
	OPENAPI_GENERATOR_VERSION=4.2.0 ./openapi-generator generate --package-name admin -i openapi.yaml -g go -o ./client
	rm -f client/go.mod client/go.sum
	go fmt ./...
	go build github.com/moov-io/customers/client
	go test ./client

.PHONY: clean
clean:
	@rm -rf client/
	@rm -f openapi-generator-cli-*.jar

dist: clean client build
ifeq ($(OS),Windows_NT)
	CGO_ENABLED=1 GOOS=windows go build -o bin/customers-windows-amd64.exe github.com/moov-io/customers/cmd/server
else
	CGO_ENABLED=1 GOOS=$(PLATFORM) go build -o bin/customers-$(PLATFORM)-amd64 github.com/moov-io/customers/cmd/server
endif

docker:
# Main Customers server Docker image
	docker build --pull -t moov/customers:$(VERSION) -f Dockerfile .
	docker tag moov/customers:$(VERSION) moov/customers:latest

release: docker AUTHORS
	go vet ./...
	go test -coverprofile=cover-$(VERSION).out ./...
	git tag -f $(VERSION)

release-push:
	docker push moov/customers:$(VERSION)
	docker push moov/customers:latest

.PHONY: cover-test cover-web
cover-test:
	go test -coverprofile=cover.out ./...
cover-web:
	go tool cover -html=cover.out

# From https://github.com/genuinetools/img
.PHONY: AUTHORS
AUTHORS:
	@$(file >$@,# This file lists all individuals having contributed content to the repository.)
	@$(file >>$@,# For how it is generated, see `make AUTHORS`.)
	@echo "$(shell git log --format='\n%aN <%aE>' | LC_ALL=C.UTF-8 sort -uf)" >> $@
