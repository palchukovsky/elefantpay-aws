VER := dev
TAG := ${VER}
ORGANIZATION := palchukovsky
PRODUCT := elefantpay-aws
CODE_REPO := github.com/${ORGANIZATION}/${PRODUCT}
IMAGES_REPO :=
MAINTAINER := local
COMMIT := local
BUILD := local
DOMAIN := elefantpay.com
EMAIL := info@${DOMAIN}
NAME := Elefantpay
AWS_PRODUCT := elefantpay
AWS_REGION := eu-central-1
AWS_ACCOUNT_ID := 102160531127
AWS_GATEWAY_ID := u46yfhcpq3
-include .env # includes only for product building, not for builders building

GO_VER := 1.14
NODE_OS_NAME := alpine
NODE_OS_TAG := 3.11
GOLANGCI_VER := 1.27.0

.PHONY: \
	help lint mock \
	install deploy \
	build build-builder build-builder-golang build-lambda-api \
	install-deps install-mock install-mock-deps
.DEFAULT_GOAL := build

ifeq (${VER}, dev)
 	LOG_SERVICE := ${PAPERTRAIL_DEV}
	DB_NAME := ${DB_NAME_DEV}
	DB_USER := ${DB_USER_DEV}
	DB_PASS := ${DB_PASS_DEV}
	LAMBDA_PREFIX := ${AWS_PRODUCT}_${VER}_
else
	LOG_SERVICE := ${PAPERTRAIL_PROD}
	DB_NAME := ${DB_NAME_PROD}
	DB_USER := ${DB_USER_PROD}
	DB_PASS := ${DB_PASS_PROD}
	LAMBDA_PREFIX := ${AWS_PRODUCT}_prod_
endif

WORKDIR := /go/src/${CODE_REPO}
GO_GET_CMD := go get -v
IMAGE_TAG := $(subst /,_,${TAG})
COMMA := ,
API_LAMBDA_PREFIX := API_
VER_DOMAIN := -dev.${DOMAIN}
LAMBDA_LFFLAGS := \
	-X '${CODE_REPO}/elefant.EmailFromName=${NAME}' \
	-X '${CODE_REPO}/elefant.EmailFromAddress=${EMAIL}' \
	-X '${CODE_REPO}/elefant.SendGridAPIKey=${SENDGRID_API_KEY}' \
	-X '${CODE_REPO}/elefant.Version=${VER}' \
	-X '${CODE_REPO}/elefant.logService=${LOG_SERVICE}' \
	-X '${CODE_REPO}/elefant.dbName=${DB_NAME}' \
	-X '${CODE_REPO}/elefant.dbUser=${DB_USER}' \
	-X '${CODE_REPO}/elefant.dbPassword=${DB_PASS}'

IMAGE_TAG_BUILDER_GOLANG := ${IMAGES_REPO}${PRODUCT}.golang:${GO_VER}-${NODE_OS_NAME}${NODE_OS_TAG}
IMAGE_TAG_BUILDER_BUILDER := ${IMAGES_REPO}${PRODUCT}.builder:${IMAGE_TAG}


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
	$(MAKE) -f ./Makefile ${1}
endef

help: ## Show this help.
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' ${MAKEFILE_LIST} | sort | awk 'BEGIN {FS = ":.*?## "};	{printf "\033[36m%-16s\033[0m %s\n", $$1, $$2}'


install: ## Deploy current sources. Uses .env file wich has to have vars.
	@$(call echo_start)
	docker run --env-file .env --rm ${IMAGE_TAG_BUILDER_BUILDER} /bin/sh -c \
		"cd ${WORKDIR} && make deploy"
	@$(call echo_success)


install-deps: ## Install package and all dependencies.
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
	-docker image rm ${IMAGE_TAG_BUILDER_BUILDER}
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
	go mod tidy
	$(call make_target,build-builder-golang)
	@$(call echo_success)

build-builder-golang: ## Build docker golang base node image.
	@$(call echo_start)
	-docker image rm ${IMAGE_TAG_BUILDER_GOLANG}
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

define zip-lambda
	zip --junk-paths bin/${VER}/lambda/${1}.zip bin/${VER}/lambda/${1}/hello
endef
define build-lambda
	GOOS=linux go build \
		-ldflags="${LAMBDA_LFFLAGS}" \
		-o bin/${VER}/lambda/${1}/hello \
		cmd/lambda/${1}/main.go
  $(call zip-lambda,${1})
endef
define build-api-lambda
	GOOS=linux go build \
	 	-ldflags="-X 'main.lambdaName=${1}' ${LAMBDA_LFFLAGS}" \
		-o bin/${VER}/lambda/api/${1}/hello \
		cmd/lambda/api/lambda/main.go
	$(call zip-lambda,api/${1})
endef
define deploy-lambda
	-aws lambda delete-function \
		--function-name ${LAMBDA_PREFIX}${2} \
		--region ${AWS_REGION} \
		--output text
	aws lambda create-function \
		--function-name ${LAMBDA_PREFIX}${2} \
		--runtime go1.x \
		--zip-file fileb://bin/${VER}/lambda/${1}.zip \
		--role arn:aws:iam::${AWS_ACCOUNT_ID}:role/${AWS_PRODUCT}BackendLambda \
		--handler hello \
		--tags product=${AWS_PRODUCT},project=backend,package=${3},version=${VER},maintainer=${MAINTAINER},commit=${COMMIT},build=${BUILD} \
		--region ${AWS_REGION} \
		--output text
endef
define permit-lambda-for-gateway
	aws lambda add-permission \
		--function-name "arn:aws:lambda:${AWS_REGION}:${AWS_ACCOUNT_ID}:function:${LAMBDA_PREFIX}${1}" \
		--source-arn "arn:aws:execute-api:${AWS_REGION}:${AWS_ACCOUNT_ID}:${AWS_GATEWAY_ID}/*" \
		--principal apigateway.amazonaws.com \
		--statement-id 92fd0de9-bf15-4136-97a8-a5d7db4c9cba \
		--action lambda:InvokeFunction \
		--region ${AWS_REGION} \
		--output text
endef
define deploy-api-lambda
	$(call deploy-lambda,api/${1},${API_LAMBDA_PREFIX}${1},api)
	$(call permit-lambda-for-gateway,${API_LAMBDA_PREFIX}${1})
endef
define for-each-api-lambda
	$(call ${1},ClientCreate)
	$(call ${1},ClientLogin)
	$(call ${1},ClientLogout)
	$(call ${1},ClientConfirm)
	$(call ${1},ClientConfirmResend)
	$(call ${1},AccountList)
	$(call ${1},AccountFind)
	$(call ${1},AccountInfo)
	$(call ${1},AccountHistory)
	$(call ${1},AccountDeposit)
	$(call ${1},AccountPaymentToAccount)
	$(call ${1},AccountPaymentTax)

endef
define upload-assets
	aws s3 cp assets/html/credentials/* s3://credentials${VER_DOMAIN}/
	aws s3api put-bucket-tagging \
		--bucket credentials${VER_DOMAIN} \
		--tagging 'TagSet=[{Key=product,Value=${AWS_PRODUCT}},{Key=project,Value=backend},{Key=package,Value=website},{Key=version,Value=${VER}},{Key=maintainer,Value=${MAINTAINER}},{Key=commit,Value=${COMMIT}},{Key=build,Value=${BUILD}}]'
endef

build-lambda-api:
	@$(call echo_start)
	$(call build-lambda,test)
	$(call build-lambda,api/auth)
	$(call for-each-api-lambda,build-api-lambda)
	@$(call echo_success)

deploy:
	@$(call echo_start)
	
	$(call upload-assets)

	$(call deploy-lambda,test,Test,test)

	$(call deploy-lambda,api/auth,${API_LAMBDA_PREFIX}Authorizer,api)
	$(call permit-lambda-for-gateway,${API_LAMBDA_PREFIX}Authorizer)

	$(call for-each-api-lambda,deploy-api-lambda)

	@$(call echo_success)