.PHONY: build clean dev image lint help
.DEFAULT_GOAL := help

BINARY_NAME="aurora"
IMAGE_NAME="pplmx/aurora"
COMPOSE_SERVICE_NAME="aurora"

DOCKERFILE_PATH="./Dockerfile"
COMPOSE_PATH="./compose.yml"

# Path: starter
MAIN_GO=cmd/aurora/main.go

dep:
	go mod download

lint:
	golangci-lint run --enable-all

test:
	go test ./...

test_coverage:
	go test ./... -coverprofile=coverage.out

check:
	go fmt ./...
	go vet ./...
	goimports -w ./...

build: check test
	CGO_ENABLED=0 GOARCH=arm64 GOOS=darwin go build -o ${BINARY_NAME}-darwin ${MAIN_GO}
	CGO_ENABLED=0 GOARCH=amd64 GOOS=darwin go build -o ${BINARY_NAME}-darwin ${MAIN_GO}
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -o ${BINARY_NAME}-linux ${MAIN_GO}
	CGO_ENABLED=0 GOARCH=amd64 GOOS=windows go build -o ${BINARY_NAME}-windows ${MAIN_GO}

run:
	./${BINARY_NAME}-linux start

image:
	docker image build -f ${DOCKERFILE_PATH} -t ${IMAGE_NAME} .

start:
	docker compose -f ${COMPOSE_PATH} -p ${COMPOSE_SERVICE_NAME} up -d

stop:
	docker compose -f ${COMPOSE_PATH} -p ${COMPOSE_SERVICE_NAME} down

restart: stop start

dev: image restart

prod: image restart

clean:
	go clean
	docker compose -f ${COMPOSE_PATH} -p ${COMPOSE_SERVICE_NAME} down
	rm -f ${BINARY_NAME}-linux
	rm -f ${BINARY_NAME}-darwin
	rm -f ${BINARY_NAME}-windows

# Show help
help:
	@echo ""
	@echo "Usage:"
	@echo " make [target]"
	@echo ""
	@echo "Targets:"
	@awk '/^[a-zA-Z\-_0-9]+:/ \
	{ \
		helpMessage = match(lastLine, /^# (.*)/); \
		if (helpMessage) { \
			helpCommand = substr($$1, 0, index($$1, ":")-1); \
			helpMessage = substr(lastLine, RSTART + 2, RLENGTH); \
			printf "\033[36m%-22s\033[0m %s\n", helpCommand,helpMessage; \
		} \
	} { lastLine = $$0 }' $(MAKEFILE_LIST)
