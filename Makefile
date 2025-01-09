SHELL=bash

APP=dp-deployer
BUILD=build
BUILD_ARCH=$(BUILD)/$(GOOS)-$(GOARCH)
BIN_DIR?=.

export GOOS?=$(shell go env GOOS)
export GOARCH?=$(shell go env GOARCH)

BUILD_TIME=$(shell date +%s)
GIT_COMMIT=$(shell git rev-parse HEAD)
VERSION ?= $(shell git tag --points-at HEAD | grep ^v | head -n 1)
LDFLAGS=-ldflags "-w -s -X 'main.Version=${VERSION}' -X 'main.BuildTime=$(BUILD_TIME)' -X 'main.GitCommit=$(GIT_COMMIT)'"

all: audit test build

audit:
	go list -m all | nancy sleuth --exclude-vulnerability-file ./.nancy-ignore

build:
	@mkdir -p $(BUILD_ARCH)/$(BIN_DIR)
	go build $(LDFLAGS) -o $(BUILD_ARCH)/$(BIN_DIR)/$(APP) cmd/$(APP)/main.go

generate:
	go generate ./...

debug: build
	HUMAN_LOG=1 go run $(LDFLAGS) -race cmd/$(APP)/main.go

test:
	go test -cover -race ./...

clean:
	rm -r $(BUILD)

.PHONY: all audit build debug test generate clean

# the following is to test dp-deployer by using:
#   make deployment
# which does the following:
#   1. a container image (and pushing to ECR)
#   2. the bundle tarball (and pushing to S3)
#   3. show how you can (ansible) deploy that build to an environment (sandbox!) for testing

ONS_DP_ENV?=sandbox
AWS_REGION?=eu-west-2
CI_PROFILE=dp-ci
CI_ECR_ACCOUNT?=$(shell aws sts --profile $(CI_PROFILE) get-caller-identity --query Account --output text)
AWS_ECR_URL?=$(CI_ECR_ACCOUNT).dkr.ecr.$(AWS_REGION).amazonaws.com
S3_BUCKET?=ons-$(CI_PROFILE)-deployments
# PROFILE uses 'production' for the path to the bundle in S3, because the ansible is hardcoded to 'production'
PROFILE?=production

ONS_DP_SRC_ROOT?=..
# for the manifest (to populate the nomad plan)
DP_CONFIGS?=$(ONS_DP_SRC_ROOT)/dp-configs
# to show you which ansible file to edit
DP_SETUP?=$(ONS_DP_SRC_ROOT)/dp-setup
# to get the nomad-glue for the bundle tarball
DP_CI?=$(ONS_DP_SRC_ROOT)/dp-ci

GIT_VERSION_ISH=$(shell git describe --tags --dirty)
BUILD_TIME_YMDHMS:=$(shell date '+%Y%m%d%H%M%S')
APP_VERSION?=$(GIT_VERSION_ISH)-$(BUILD_TIME_YMDHMS)
IMAGE_TAG?=test-$(USER)-$(APP_VERSION)
IMAGE_NAME?=$(APP)
IMAGE_URL?=$(AWS_ECR_URL)/$(IMAGE_NAME)
NOMAD_PLAN=$(APP).nomad
BUNDLE_TAR=$(APP_VERSION).tar.gz

.PHONY: image-prep
image-prep:
	$(MAKE) build GOOS=linux GOARCH=amd64 IMAGE_TAG=$(IMAGE_TAG)
.PHONY: image-build
image-build: image-prep
	docker build -f Dockerfile.test-in-env --tag $(IMAGE_URL):$(IMAGE_TAG) .
	@echo
	@echo "1. To push this image and the bundle, you may need to run:"
	@echo "make image-push bundle IMAGE_TAG=$(IMAGE_TAG)"
.PHONY: image-push
image-push:
	AWS_PROFILE=$(CI_PROFILE) docker push $(IMAGE_URL):$(IMAGE_TAG)

.PHONY: deployment
deployment: image bundle
.PHONY: image
image: image-build image-push
.PHONY: bundle
bundle: bundle-build bundle-push

.PHONY: bundle-build
bundle-build:
	mkdir -p $(BUILD)/bundle
	cp -p $(DP_CI)/nomad-glue/* $(BUILD)/bundle
	nomad_json=$$(yq -o=json '.nomad.groups[]|select(.class=="management")|.profiles["$(ONS_DP_ENV)"]' $(DP_CONFIGS)/manifests/$(APP).yml);	\
	sed	-e "s/{{MANAGEMENT_TASK_COUNT}}/$$(jq   -r .count            <<<"$$nomad_json")/g"	\
		-e "s/{{MANAGEMENT_RESOURCE_CPU}}/$$(jq -r .resources.cpu    <<<"$$nomad_json")/g"	\
		-e "s/{{MANAGEMENT_RESOURCE_MEM}}/$$(jq -r .resources.memory <<<"$$nomad_json")/g"	\
		-e "s/concourse-{{REVISION}}/$(IMAGE_TAG)/g"						\
		-e 's|{{DEPLOYMENT_BUCKET}}|$(S3_BUCKET)|g'						\
		-e "s|{{ECR_URL}}|https://$(IMAGE_URL)|g"						\
		-e "s/{{PROFILE}}/$(PROFILE)/g"								\
		-e "s/{{RELEASE}}/$(APP_VERSION)/g"							\
			< $(NOMAD_PLAN) > $(BUILD)/bundle/$(APP).nomad
	tar -zcvf $(BUILD)/$(BUNDLE_TAR) -C $(BUILD)/bundle .
.PHONY: bundle-push
bundle-push:
	aws s3 cp --profile=$(CI_PROFILE) $(BUILD)/$(BUNDLE_TAR) s3://$(S3_BUCKET)/$(APP)/$(PROFILE)/$(BUNDLE_TAR)
	@echo "You now need to edit and run ansible"
	@echo "1. in $(DP_SETUP)/ansible/roles/bootstrap-deployer/defaults/main.yml"
	@echo "   amend the 'dp_deployer_version' line to read 'dp_deployer_version: $(APP_VERSION)'"
	@echo "vim +/dp_deployer_version $(DP_SETUP)/ansible/roles/bootstrap-deployer/defaults/main.yml"
	@echo "2. You then need to run the ansible:"
	@echo "ansible-playbook --vault-id=$(ONS_DP_ENV)@.$(ONS_DP_ENV).pass -i inventories/$(ONS_DP_ENV) bootstrap-deployer.yml"
