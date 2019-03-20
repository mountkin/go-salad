.PHONY: build clean deploy image destroy

PKG := github.com/mountkin/go-salad/webhook-relay
IMAGE_NAME := mountkin/webhook-relay

SHELL := /bin/bash
UID := $(shell id -u)
GID := $(shell id -g)

default: build

build: image
	[ -d bin ] || mkdir bin
	docker run --user $(UID):$(GID) --rm -v $(CURDIR)/bin:/target $(IMAGE_NAME) cp /bin/webhook /target/

image: 
	docker build -t $(IMAGE_NAME) --force-rm .

clean:
	rm -rf ./bin

deploy: clean build
	@if [ -n "$(GITHUB_WEBHOOK_SECRET)" ]; then \
		sed -i 's/__GITHUB_WEBHOOK_SECRET__/$(GITHUB_WEBHOOK_SECRET)/g' serverless.yml; \
	fi
	sls deploy -v

destroy: clean
	sls remove -v
