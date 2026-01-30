# Copyright 2017-2026 Adobe. All rights reserved.
# This file is licensed to you under the Apache License, Version 2.0 (the "License"); 
# you may not use this file except in compliance with the License. You may obtain a copy
# of the License at http://www.apache.org/licenses/LICENSE-2.0

# Unless required by applicable law or agreed to in writing, software distributed under
# the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR REPRESENTATIONS 
# OF ANY KIND, either express or implied. See the License for the specific language
# governing permissions and limitations under the License.

SHELL := /bin/bash
.PHONY: build test test-unit test-accept clean help fmt lint vet check build-local ci all

# Project configuration
SERVICE_NAME := butler
BUTLER_VERSION := $(shell cat VERSION)
VERSION := v$(BUTLER_VERSION)

# Git information
SHA := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go configuration
GO := go
PKGS := $(shell $(GO) list ./... | grep -v vendor)

# Docker configuration
DOCKERHUB_USER ?= $(shell echo "$$DOCKERHUB_USER")
REGISTRY ?= 
REGISTRY_PREFIX ?= 

# Export variables for docker-bake.hcl
export VERSION
export SHA
export REGISTRY
export REGISTRY_PREFIX

########################################################################################################################
# Default target
########################################################################################################################

default: ci

########################################################################################################################
# CI/CD targets using docker buildx bake
########################################################################################################################

ci: build
	@echo "CI build complete"

all: build test
	@echo "Full build and test complete"

# Build the butler image using docker buildx bake
build:
	@echo "> Building butler container image"
	docker buildx bake butler

# Build all images (butler + debug)
build-all:
	@echo "> Building all container images"
	docker buildx bake all

# Build multi-platform images
build-multiplatform:
	@echo "> Building multi-platform images"
	docker buildx bake butler-multiplatform

########################################################################################################################
# Test targets using docker buildx bake
########################################################################################################################

test: test-unit test-accept
	@echo "All tests passed"

test-unit:
	@echo "> Running unit tests"
	docker buildx bake test-unit
	@echo "> Coverage reports available in out/coverage/"

test-accept:
	@echo "> Running acceptance tests"
	docker buildx bake test-accept

# Interactive test containers for debugging
enter-test-unit:
	@echo "> Building interactive unit test container"
	docker buildx bake test-unit-interactive
	docker run -it --rm \
		-v $(PWD):/src \
		-v /tmp/coverage:/tmp/coverage \
		butler-test-unit:dev /bin/sh

enter-test-accept:
	@echo "> Building interactive acceptance test container"
	docker buildx bake test-accept-interactive
	docker run -it --rm \
		-v $(PWD)/files/certs:/certs \
		-v $(PWD)/files/tests:/www \
		butler-test-accept:dev /bin/sh

########################################################################################################################
# Local development targets (no Docker)
########################################################################################################################

build-local: fmt
	@echo "> Building local butler binary"
	$(GO) build -ldflags "-X main.version=$(VERSION)" -o butler cmd/butler/main.go

test-local:
	@echo "> Running local tests"
	$(GO) test $(PKGS) -v

fmt:
	@echo "> Formatting go files"
	@find . -path ./vendor -prune -o -name '*.go' -print | xargs gofmt -s -w

lint:
	@echo "> Linting go files"
	@golint $(PKGS) 2>/dev/null || echo "golint not installed, skipping"

vet:
	@echo "> Vetting go files"
	$(GO) vet $(PKGS)

check: fmt vet lint

########################################################################################################################
# Publishing
########################################################################################################################

# Push to configured registry (set REGISTRY and REGISTRY_PREFIX env vars)
push: build
	@echo "> Pushing butler image to registry"
	@if [ -z "$(REGISTRY)" ]; then \
		echo "Error: REGISTRY environment variable not set"; \
		echo "Usage: REGISTRY=myregistry.com REGISTRY_PREFIX=docker-local/ make push"; \
		exit 1; \
	fi
	docker buildx bake butler --push

# Push all images (butler + debug) to configured registry
push-all: build-all
	@echo "> Pushing all images to registry"
	@if [ -z "$(REGISTRY)" ]; then \
		echo "Error: REGISTRY environment variable not set"; \
		exit 1; \
	fi
	docker buildx bake default --push

# Legacy DockerHub push (for backwards compatibility)
push-dockerhub: build
	@printf "Enter DockerHub "
	@docker login -u $(DOCKERHUB_USER)
	docker tag butler:$(VERSION) $(DOCKERHUB_USER)/$(SERVICE_NAME):$(BUTLER_VERSION)
	docker push $(DOCKERHUB_USER)/$(SERVICE_NAME):$(BUTLER_VERSION)

########################################################################################################################
# Utility targets
########################################################################################################################

clean:
	@echo "> Cleaning build artifacts"
	rm -f butler
	rm -rf out/
	docker rmi butler:$(VERSION) butler:latest butler-debug:$(VERSION) butler-debug:latest 2>/dev/null || true

# Show what docker buildx bake would do
bake-print:
	docker buildx bake --print

########################################################################################################################
# Local service targets for development/testing
########################################################################################################################

run:
	$(GO) run -ldflags "-X main.version=$(VERSION)" cmd/butler/main.go \
		-config.path http://localhost/butler/config/butler.toml \
		-config.retrieve-interval 10 \
		-log.level debug

start-etcd:
	@docker run --rm -it --name=etcd -d \
		-p 4001:4001 -p 2379:2379 -p 2380:2380 \
		-v /tmp:/tmp \
		quay.io/coreos/etcd:v3.2.17 etcd \
		--name etcd \
		--initial-cluster-state new \
		--advertise-client-urls http://127.0.0.1:2379,http://127.0.0.1:4001 \
		--listen-client-urls http://0.0.0.0:2379,http://0.0.0.0:4001 \
		--initial-cluster-token etcd-cluster-1 \
		--initial-cluster etcd=http://127.0.0.1:2380 \
		--initial-advertise-peer-urls http://127.0.0.1:2380

start-prometheus:
	@docker run --rm -it --name=prometheus -d \
		-p 9090:9090 \
		-v /opt/prometheus:/etc/prometheus \
		prom/prometheus:v1.8.2 \
		-config.file=/etc/prometheus/prometheus.yml \
		-storage.local.path=/prometheus \
		-storage.local.memory-chunks=104857

start-prometheus2:
	@docker run --rm -it --name=prometheus -d \
		-p 9090:9090 \
		-v /opt/prometheus:/etc/prometheus \
		prom/prometheus:v2.1.0 \
		--config.file=/etc/prometheus/prometheus.yml \
		--storage.tsdb.path=/prometheus

start-alertmanager:
	@docker run --rm -it --name=alertmanager -d \
		-p 9093:9093 \
		-v /opt/butler/alertmanager:/etc/alertmanager \
		prom/alertmanager:v0.13.0 \
		--config.file=/etc/alertmanager/alertmanager.yml

start-am-prom: start-prometheus start-alertmanager

stop-etcd:
	@docker stop etcd

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

########################################################################################################################
# Help
########################################################################################################################

help:
	@printf "Butler Build System\n"
	@printf "==================\n\n"
	@printf "Build targets:\n"
	@printf "  make                    Build butler image (CI default)\n"
	@printf "  make build              Build butler image using docker buildx bake\n"
	@printf "  make build-all          Build all images (butler + debug)\n"
	@printf "  make build-local        Build local binary (no Docker)\n"
	@printf "  make build-multiplatform Build for amd64 and arm64\n"
	@printf "\n"
	@printf "Test targets:\n"
	@printf "  make test               Run all tests (unit + acceptance)\n"
	@printf "  make test-unit          Run unit tests in container\n"
	@printf "  make test-accept        Run acceptance tests in container\n"
	@printf "  make test-local         Run tests locally (no Docker)\n"
	@printf "  make enter-test-unit    Interactive unit test container\n"
	@printf "  make enter-test-accept  Interactive acceptance test container\n"
	@printf "\n"
	@printf "Code quality:\n"
	@printf "  make fmt                Format Go code\n"
	@printf "  make lint               Run linter\n"
	@printf "  make vet                Run go vet\n"
	@printf "  make check              Run fmt, vet, and lint\n"
	@printf "\n"
	@printf "Publishing:\n"
	@printf "  make push-dockerhub     Push to DockerHub\n"
	@printf "\n"
	@printf "Utility:\n"
	@printf "  make clean              Remove build artifacts\n"
	@printf "  make bake-print         Show docker buildx bake plan\n"
	@printf "\n"
	@printf "Local services (for testing):\n"
	@printf "  make run                Run butler locally\n"
	@printf "  make start-etcd         Start local etcd\n"
	@printf "  make start-prometheus   Start Prometheus v1\n"
	@printf "  make start-prometheus2  Start Prometheus v2\n"
	@printf "  make start-alertmanager Start Alertmanager\n"
	@printf "  make stop-*             Stop respective service\n"
