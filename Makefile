GNORED_FOLDER=.ignore
COVERAGE_FILE=$(IGNORED_FOLDER)/coverage.out

.PHONY: all install build lint test cover cover-html clean tools-lint tools help

all: tools install lint test build ## Start all: tools install lint test build

##
## Building
##
install: ## Download and install go mod
	@go mod download

build: ## Build App
	go build ./...

##
## Quality Code
##
lint: ## Lint
	@golangci-lint run

test: mock ## Test
	@mkdir -p ${IGNORED_FOLDER}
	@go test -gcflags=-l -count=1 -race -coverprofile=${COVERAGE_FILE} -covermode=atomic ./...

cover: ## Cover
	@if [ ! -e ${COVERAGE_FILE} ]; then \
		echo "Error: ${COVERAGE_FILE} doesn't exists. Please run \`make test\` then retry."; \
		exit 1; \
	fi
	@go tool cover -func=${COVERAGE_FILE}


cover-html: ## Cover html
	@if [ ! -e ${COVERAGE_FILE} ]; then \
		echo "Error: ${COVERAGE_FILE} doesn't exists. Please run \`make test\` then retry."; \
		exit 1; \
	fi
	@go tool cover -html=${COVERAGE_FILE}

##
## Cleanning
##
clean: ## Clean
	@rm -rf ${COVERAGE_FILE}


##
## Tooling
##
tools-lint: ## Install go lint dependencies
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

tools: tools-lint 

##
## Help
##
help: ## Help
	@grep -E '(^[a-zA-Z_-]+:.*?#.*$$)|(^#)' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?# "}; {printf "\033[32m%-30s\033[0m %s\n", $$1, $$2}' | sed -e 's/\[32m#/[33m/'