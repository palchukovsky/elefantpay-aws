TAG := dev
ORGANIZATION = palchukovsky
PRODUCT = elefantpay-aws
CODE_REPO = github.org/palchukovsky/elefantpay-aws
IMAGES_REPO =
MAINTAINER = local
COMMIT = local
BUILD = local
AWS_REGION = eu-central-1

GO_VER = 1.14
NODE_OS_NAME = alpine
NODE_OS_TAG = 3.11
GOLANGCI_VER = 1.27.0

.PHONY: \
	help lint mock \
	deploy deploy-lambda-api \
	build build-builder build-builder-golang build-lambda-api \
	install-mock install-mock-deps
.DEFAULT_GOAL := help

WORKDIR = /go/src/${CODE_REPO}
THIS_FILE := $(lastword ${MAKEFILE_LIST})
GO_GET_CMD = go get -v
IMAGE_TAG := $(subst /,_,${TAG})
COMMA := ,

IMAGE_TAG_BUILDER_GOLANG = ${IMAGES_REPO}${PRODUCT}.golang:${GO_VER}-${NODE_OS_NAME}${NODE_OS_TAG}
IMAGE_TAG_BUILDER_BUILDER = ${IMAGES_REPO}${PRODUCT}.builder:${GO_VER}-${NODE_OS_NAME}${NODE_OS_TAG}


define echo_start
	@echo ================================================================================
	@echo :
	@echo : START: $(@)
	@echo :
endef
define echo_success
	@echo :
	@echo : SUCCESS: $(@)
	@echo :
	@echo ================================================================================
endef

define make_target
	$(MAKE) -f ./$(THIS_FILE) $(1)
endef

help: ## Show this help.
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' ${MAKEFILE_LIST} | sort | awk 'BEGIN {FS = ":.*?## "};	{printf "\033[36m%-16s\033[0m %s\n", $$1, $$2}'


deploy:  ## Deploy current sources. Uses .env file wich has to have vars AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY
	@$(call echo_start)
	$(call make_target,build)
	docker run --env-file .env --rm ${IMAGE_TAG_BUILDER_BUILDER} /bin/sh -c \
		"cd ${WORKDIR} && make deploy-lambda-api"
	@$(call echo_success)


install: ## Install package and all dependencies.
	@$(call echo_start)
	${GO_GET_CMD} ./...
	$(call make_target,install-mock)
	go mod tidy
	@$(call echo_success)

install-mock: ## Install mock compilator and generate mock.
	@$(call echo_start)
	$(call make_target,install-mock-deps)
	$(call make_target,mock)
	@$(call echo_success)

install-mock-deps: ## Install mock compilator components.
	@$(call echo_start)
	${GO_GET_CMD} github.com/stretchr/testify/assert
	${GO_GET_CMD} github.com/golang/mock/gomock
	${GO_GET_CMD} github.com/golang/mock/mockgen
	@$(call echo_success)


build: ## Build docker builder image with actual project sources.
	@$(call echo_start)
	docker build --file "./build/builder/builder.Dockerfile" \
		--build-arg GOLANG=${IMAGE_TAG_BUILDER_GOLANG} \
		--build-arg WORKDIR=${WORKDIR} \
		--label "Maintainer=${MAINTAINER}" \
		--label "Commit=${COMMIT}" \
		--label "Build=${BUILD}" \
		--tag ${IMAGE_TAG_BUILDER_BUILDER} \
		./
	@$(call echo_success)

build-builder: ## Build all docker images for builder.
	@$(call echo_start)
	$(call make_target,build-builder-golang)
	@$(call echo_success)

build-builder-golang: ## Build docker golang base node image.
	@$(call echo_start)
	docker build --file "./build/builder/golang.Dockerfile" \
		--build-arg GOLANG_TAG=${GO_VER}-${NODE_OS_NAME}${NODE_OS_TAG} \
		--build-arg GOLANGCI_VER=${GOLANGCI_VER} \
		--build-arg AWSCLI_VERSION=${AWSCLI_VERSION} \
		--build-arg WORKDIR=${WORKDIR} \
		--label "Maintainer=${MAINTAINER}" \
		--label "Commit=${COMMIT}" \
		--label "Build=${BUILD}" \
		--tag ${IMAGE_TAG_BUILDER_GOLANG} \
		./
	@$(call echo_success)


lint: ## Run linter.
	@$(call echo_start)
	golangci-lint run -v ./...
	@$(call echo_success)


mock: ## Generate mock interfaces for unit-tests.


define build-lambda
	GOOS=linux go build -o bin/$(1)/handler $(1)/main.go
  zip --junk-paths bin/$(1).zip bin/$(1)/handler
endef
define configure-deploy
endef
define deploy-lambda
	aws lambda update-function-code \
		--function-name $(2) \
		--zip-file fileb://bin/$(1).zip \
		--region ${AWS_REGION} \
		--output text
endef

build-lambda-api:
	@$(call echo_start)
	$(call build-lambda,lambda/api/account/create)
	$(call build-lambda,lambda/api/account/login)
	@$(call echo_success)

deploy-lambda-api:
	@$(call echo_start)
	$(call configure-deploy)
	$(call deploy-lambda,lambda/api/account/create,APIAccountCreate)
	$(call deploy-lambda,lambda/api/account/login,APIAccountLogin)
	@$(call echo_success)