export SERVICE_NAME=butler
export BUILDER_TAG?=$(or $(sha),$(SERVICE_NAME)-builder)
export TESTER_TAG?=$(or $(sha),$(SERVICE_NAME)-tester)

IMAGE_TAG=$(SERVICE_NAME)-img

GO:=go
pkgs=$(shell $(GO) list ./... | egrep -v "(vendor)")

export ARTIFACTORY_USER=$(shell echo "$$ARTIFACTORY_USER")
export ARTIFACTORY_REPO=butler
export ARTIFACTORY_VERSION=1.1.5
export VERSION=v$(ARTIFACTORY_VERSION)
export ARTIFACTORY_PROD_HOST=docker-ethos-core-univ-release.dr-uw2.adobeitc.com
export ARTIFACTORY_DEV_HOST=docker-ethos-core-univ-dev.dr-uw2.adobeitc.com

default: ci

ci: build
	@echo "Success"

build:
	@docker build --build-arg VERSION=$(VERSION) -t $(BUILDER_TAG) -f Dockerfile-build .
	@docker run -v m2:/root/.m2 -v `pwd`:/build $(BUILDER_TAG) cp /root/butler/butler /build
	@docker build -t $(IMAGE_TAG) .

build-local:
	@$(GO) fmt $(pkgs)
	@$(GO) build

pre-deploy-build: test

post-deploy-build:
	@echo "Nothing is defined in post-deploy-build step"

test:
	@docker build --build-arg VERSION=$(VERSION) -t $(TESTER_TAG) -f Dockerfile-test .
	@docker run -i $(TESTER_TAG)

enter-test:
	@./files/enter_test_container.sh

test-local:
	@go test $(pkgs) -check.vv -v

build-$(ARTIFACTORY_REPO):
	@docker build --build-arg VERSION=$(VERSIO) -t $(ARTIFACTORY_REPO):$(ARTIFACTORY_VERSION) .

push-$(ARTIFACTORY_REPO)-release: DOCKER_IMAGE_ID = $(shell docker images -q $(ARTIFACTORY_REPO):$(ARTIFACTORY_VERSION))
push-$(ARTIFACTORY_REPO)-release: build-$(ARTIFACTORY_REPO)
	@printf "Enter DockerHub "
	@docker login -u $(ARTIFACTORY_USER) $(ARTIFACTORY_PROD_HOST)
	docker tag $(DOCKER_IMAGE_ID) $(ARTIFACTORY_PROD_HOST)/ethos/$(ARTIFACTORY_REPO):$(ARTIFACTORY_VERSION)
	docker push $(ARTIFACTORY_PROD_HOST)/ethos/$(ARTIFACTORY_REPO):$(ARTIFACTORY_VERSION)

push-$(ARTIFACTORY_REPO)-dev: DOCKER_IMAGE_ID = $(shell docker images -q $(ARTIFACTORY_REPO):$(ARTIFACTORY_VERSION))
push-$(ARTIFACTORY_REPO)-dev: build-$(ARTIFACTORY_REPO)
	@printf "Enter DockerHub "
	@docker login -u $(ARTIFACTORY_USER) $(ARTIFACTORY_DEV_HOST)
	docker tag $(DOCKER_IMAGE_ID) $(ARTIFACTORY_DEV_HOST)/ethos/$(ARTIFACTORY_REPO):$(ARTIFACTORY_VERSION)
	docker push $(ARTIFACTORY_DEV_HOST)/ethos/$(ARTIFACTORY_REPO):$(ARTIFACTORY_VERSION)

push-butler-dockerhub: DOCKER_IMAGE_ID = $(shell docker images -q $(ARTIFACTORY_REPO):$(ARTIFACTORY_VERSION))
push-butler-dockerhub: build-$(ARTIFACTORY_REPO)
	@printf "Enter DockerHub "
	@docker login -u $(ARTIFACTORY_USER)
	docker tag $(DOCKER_IMAGE_ID) $(ARTIFACTORY_USER)/$(ARTIFACTORY_REPO):$(ARTIFACTORY_VERSION)
	docker push $(ARTIFACTORY_USER)/$(ARTIFACTORY_REPO):$(ARTIFACTORY_VERSION)

help:
	@printf "Usage:\n\n"
	@printf "make\t\t\t\tBuilds butler, for use in CI.\n"
	@printf "make build-local\t\tBuilds a local binary of butler.\n"
	@printf "make build-$(ARTIFACTORY_REPO)\t\tBuilds butler locally, for use in pushing to artifactory.\n"
	@printf "make push-$(ARTIFACTORY_REPO)-dev\t\tPushes butler to $(ARTIFACTORY_DEV_HOST).\n"
	@printf "make push-$(ARTIFACTORY_REPO)-release\tPushes butler to $(ARTIFACTORY_PROD_HOST).\n"
	@printf "make push-butler-dockerhub\tPushes butler to DockerHub (If necessary).\n"
	@printf "make run\t\t\tRun butler on local system.\n"
	@printf "make start-alertmanager\t\tRun a local alertmanager instance for testing.\n"
	@printf "make start-prometheus\t\tRun a local prometheus instance for testing.\n"
	@printf "make start-am-prom\t\tRun a local alertmanager and prometheus instance for testing.\n"
	@printf "make stop-alertmanager\t\tStop the local test alertmanager instance.\n"
	@printf "make stop-prometheus\t\tStop the local test prometheus instance.\n"
	@printf "make stop-am-prom\t\tStop the local test alertmanager and prometheus instance.\n"
	@printf "make prometheus-logs\t\tTail the logs of the test prometheus instance.\n"
	@printf "make alertmanager-logs\t\tTail the logs of the test prometheus instance.\n"

run:
	$(GO) run -ldflags "-X main.version=$(VERSION)" butler.go -config.path http://localhost/butler/config/butler.toml -config.retrieve-interval 10 -log.level debug

start-prometheus:
	@docker run --rm -it --name=prometheus -d -p 9090:9090 -v /opt/prometheus:/etc/prometheus prom/prometheus:v1.8.2 -config.file=/etc/prometheus/prometheus.yml -storage.local.path=/prometheus -storage.local.memory-chunks=104857

start-alertmanager:
	@docker run --rm -it --name=alertmanager -d -p 9093:9093 -v /opt/alertmanager:/etc/alertmanager prom/alertmanager -config.file=/etc/alertmanager/alertmanager.yml

start-am-prom:
	@docker run --rm -it --name=prometheus -d -p 9090:9090 -v /opt/prometheus:/etc/prometheus prom/prometheus -config.file=/etc/prometheus/prometheus.yml -storage.local.path=/prometheus -storage.local.memory-chunks=104857
	@docker run --rm -it --name=alertmanager -d -p 9093:9093 -v /opt/alertmanager:/etc/alertmanager prom/alertmanager -config.file=/etc/alertmanager/alertmanager.yml

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
