# Copyright 2017 Adobe. All rights reserved.
# This file is licensed to you under the Apache License, Version 2.0 (the "License"); 
# you may not use this file except in compliance with the License. You may obtain a copy
# of the License at http://www.apache.org/licenses/LICENSE-2.0

# Unless required by applicable law or agreed to in writing, software distributed under
# the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR REPRESENTATIONS 
# OF ANY KIND, either express or implied. See the License for the specific language
# governing permissions and limitations under the License.

SHELL=/bin/bash
.PHONY: test
export SERVICE_NAME=butler
export BUILDER_TAG?=$(or $(sha),$(SERVICE_NAME)-builder)
export UNIT_TESTER_TAG?=$(or $(sha),$(SERVICE_NAME)-unittester)
export ACCEPT_TESTER_TAG?=$(or $(sha),$(SERVICE_NAME)-accepttester)

IMAGE_TAG=$(SERVICE_NAME)-img

GO:=go
pkgs=$(shell $(GO) list ./... | egrep -v "(vendor)")

export DOCKERHUB_USER=$(shell echo "$$DOCKERHUB_USER")
export BUTLER_VERSION=1.3.0
export VERSION=v$(BUTLER_VERSION)

default: ci

ci: build
	@echo "Success"

all: build test push-dockerhub

build:
	@echo "> building container butler binary"
	@docker build --build-arg VERSION=$(VERSION) -t $(BUILDER_TAG) -f files/Dockerfile-build .
	@docker run -v m2:/root/.m2 -v `pwd`:/build $(BUILDER_TAG) cp /root/butler/butler /build
	@docker build -t $(IMAGE_TAG) .

fmt:
	@echo "> formatting go files"
	@find . -path ./vendor -prune -o -name '*.go' -print | xargs gofmt -s -w

build-local: fmt
	@echo "> building local butler binary"
	@$(GO) build -ldflags "-X main.version=$(VERSION)" -o butler cmd/butler/main.go

pre-deploy-build: test-unit

post-deploy-build:
	@echo "Nothing is defined in post-deploy-build step"

test: test-unit test-accept
test-unit:
	@docker build --build-arg VERSION=$(VERSION) -t $(UNIT_TESTER_TAG) -f files/Dockerfile-testunit .
	@docker run -i $(UNIT_TESTER_TAG)

test-accept:
	@docker build --build-arg VERSION=$(VERSION) -t $(ACCEPT_TESTER_TAG) -f files/Dockerfile-testaccept .
	@docker run -v `pwd`/files/certs:/certs -v `pwd`/files/tests:/www --env-file <(env | egrep "(^BUT|AWS)") -i $(ACCEPT_TESTER_TAG)

enter-test-unit:
	@./files/enter_unit_test_container.sh

enter-test-accept:
	@./files/enter_accept_test_container.sh

test-local:
	@go test $(pkgs) -check.vv -v

build-$(SERVICE_NAME):
	@docker build --build-arg VERSION=$(VERSION) -t $(SERVICE_NAME):$(BUTLER_VERSION) .

push-dockerhub: DOCKER_IMAGE_ID = $(shell docker images -q $(SERVICE_NAME):$(BUTLER_VERSION))
push-dockerhub: build-$(SERVICE_NAME)
	@printf "Enter DockerHub "
	@docker login -u $(DOCKERHUB_USER)
	docker tag $(DOCKER_IMAGE_ID) $(DOCKERHUB_USER)/$(SERVICE_NAME):$(BUTLER_VERSION)
	docker push $(DOCKERHUB_USER)/$(SERVICE_NAME):$(BUTLER_VERSION)

help:
	@printf "Usage:\n\n"
	@printf "make\t\t\t\tBuilds butler, for use in CI.\n"
	@printf "make build-local\t\tBuilds a local binary of butler.\n"
	@printf "make build-$(SERVICE_NAME)\t\tBuilds butler locally, for use in pushing to artifactory.\n"
	@printf "make push-dockerhub\t\tPushes butler to DockerHub (If necessary).\n"
	@printf "make run\t\t\tRun butler on local system.\n"
	@printf "make start-alertmanager\t\tRun a local alertmanager instance for testing.\n"
	@printf "make start-prometheus\t\tRun a local prometheus v1 instance for testing.\n"
	@printf "make start-prometheus2\t\tRun a local prometheus v2 instance for testing.\n"
	@printf "make start-am-prom\t\tRun a local alertmanager and prometheus instance for testing.\n"
	@printf "make stop-alertmanager\t\tStop the local test alertmanager instance.\n"
	@printf "make stop-prometheus\t\tStop the local test prometheus instance.\n"
	@printf "make stop-am-prom\t\tStop the local test alertmanager and prometheus instance.\n"
	@printf "make prometheus-logs\t\tTail the logs of the test prometheus instance.\n"
	@printf "make alertmanager-logs\t\tTail the logs of the test prometheus instance.\n"

run:
	$(GO) run -ldflags "-X main.version=$(VERSION)" cmd/butler/main.go -config.path http://localhost/butler/config/butler.toml -config.retrieve-interval 10 -log.level debug

start-prometheus:
	@docker run --rm -it --name=prometheus -d -p 9090:9090 -v /opt/prometheus:/etc/prometheus prom/prometheus:v1.8.2 -config.file=/etc/prometheus/prometheus.yml -storage.local.path=/prometheus -storage.local.memory-chunks=104857

start-prometheus2:
	@docker run --rm -it --name=prometheus -d -p 9090:9090 -v /opt/prometheus:/etc/prometheus prom/prometheus:v2.1.0 --config.file=/etc/prometheus/prometheus.yml --storage.tsdb.path=/prometheus

start-alertmanager:
	@docker run --rm -it --name=alertmanager -d -p 9093:9093 -v /opt/butler/alertmanager:/etc/alertmanager prom/alertmanager:v0.13.0 --config.file=/etc/alertmanager/alertmanager.yml

start-am-prom: start-prometheus start-alertmanager
#	@docker run --rm -it --name=prometheus -d -p 9090:9090 -v /opt/prometheus:/etc/prometheus prom/prometheus -config.file=/etc/prometheus/prometheus.yml -storage.local.path=/prometheus -storage.local.memory-chunks=104857
#	@docker run --rm -it --name=alertmanager -d -p 9093:9093 -v /opt/alertmanager:/etc/alertmanager prom/alertmanager -config.file=/etc/alertmanager/alertmanager.yml

stop-prometheus:
	@docker stop prometheus

stop-alertmanager:
	@docker stop alertmanager

stop-am-prom:
	@docker stop prometheus
	@docker stop alertmanager

prometheus-logs:
	@docker logs -f prometheus

alertmanager-logs:
	@docker logs -f alertmanager
