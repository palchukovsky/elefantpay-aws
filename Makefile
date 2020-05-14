.PHONY: help install build
.DEFAULT_GOAL := help

THIS_FILE := $(lastword ${MAKEFILE_LIST})
GO_GET_CMD = go get -v

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

help: ## Show this help.
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' ${MAKEFILE_LIST} | sort | awk 'BEGIN {FS = ":.*?## "};	{printf "\033[36m%-16s\033[0m %s\n", $$1, $$2}'

install: ## Install package and all dependencies.
	@$(call echo_start)
	go get github.com/aws/aws-lambda-go/cmd/build-lambda-zip
	@$(call echo_success)

build:  ## Build all from actual local source. GOOS=linux
	@$(call echo_start)
	go build -o account_create lambda/api/account/create/main.go
	build-lambda-zip -output account_create.zip account_create
	go build -o account_login lambda/api/account/login/main.go
	build-lambda-zip -output account_login.zip account_login
	@$(call echo_success)