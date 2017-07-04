SERVICE_NAME=butler
BUILDER_TAG?=$(or $(sha),$(SERVICE_NAME)-builder)
TESTER_TAG?=$(or $(sha),$(SERVICE_NAME)-tester)

IMAGE_TAG=$(SERVICE_NAME)-img

GO:=go
pkgs=$(shell $(GO) list ./... | egrep -v "(vendor)")

DOCKERHUB_USER=matthsmi
DOCKERHUB_REPO=butler
DOCKERHUB_VERSION=v0.3.4

default: ci

ci: build
	@echo "Success"

build:
	@docker build -t $(BUILDER_TAG) -f Dockerfile-build .
	@docker run -v m2:/root/.m2 -v `pwd`:/build $(BUILDER_TAG) cp /root/butler/butler /build
	@docker build -t $(IMAGE_TAG) .

pre-deploy-build:
	@docker build -t $(TESTER_TAG) -f Dockerfile-test .
	@docker run -it --rm $(TESTER_TAG)

post-deploy-build:
	@echo "Nothing is defined in post-deploy-build step"

build-$(DOCKERHUB_REPO):
	@docker build -t $(DOCKERHUB_REPO):$(DOCKERHUB_VERSION) .

push-$(DOCKERHUB_REPO): DOCKER_IMAGE_ID = $(shell docker images -q $(DOCKERHUB_REPO):$(DOCKERHUB_VERSION))
push-$(DOCKERHUB_REPO):
	@printf "Enter DockerHub "
	@docker login -u $(DOCKERHUB_USER)
	docker tag $(DOCKER_IMAGE_ID) $(DOCKERHUB_USER)/$(DOCKERHUB_REPO):latest
	docker push $(DOCKERHUB_USER)/$(DOCKERHUB_REPO):latest
	docker tag $(DOCKER_IMAGE_ID) $(DOCKERHUB_USER)/$(DOCKERHUB_REPO):$(DOCKERHUB_VERSION)
	docker push $(DOCKERHUB_USER)/$(DOCKERHUB_REPO):$(DOCKERHUB_VERSION)

run:
	$(GO) run butler.go -config.url http://git1.dev.or1.adobe.net/cgit/adobe-platform/ethos-monitoring/plain/oncluster -config.cluster-id ethos01-dev-or1 -config.schedule-interval 1
