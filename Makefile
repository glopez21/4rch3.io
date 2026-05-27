BINARY=portal
BUILD_DIR=build
COMMIT=$(shell git describe --always --dirty 2>/dev/null || echo "dev")
DATE=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS=-ldflags "-X main.commit=$(COMMIT) -X main.date=$(DATE)"

.PHONY: all build test vet lint clean run docker-build docker-run

all: vet test build

build:
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY) ./cmd/server

test:
	go test ./... -v -count=1

vet:
	go vet ./...

lint:
	which golangci-lint >/dev/null 2>&1 && golangci-lint run ./... || echo "install golangci-lint: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"

clean:
	rm -rf $(BUILD_DIR)

run:
	go run ./cmd/server

docker-build:
	docker build -t 4rch3/portal:latest -t 4rch3/portal:$(COMMIT) .

docker-run:
	docker compose up -d

docker-logs:
	docker compose logs -f
