# VERSION := $(shell echo $(shell git describe --tags) | sed 's/^v//')
# COMMIT := $(shell git log -1 --format='%H')

# build_tags = cppbtree

# ldflags += -X github.com/smartbch/smartbch/app.GitCommit=$(COMMIT) \
# 		  -X github.com/cosmos/cosmos-sdk/version.GitTag=$(VERSION)

# BUILD_FLAGS := -tags "$(build_tags)" -ldflags '$(ldflags)'

# build: go.sum
# ifeq ($(OS), Windows_NT)
# 	go build -mod=readonly $(BUILD_FLAGS) -o build/smartbchd.exe ./cmd/smartbchd
# else
# 	go build -mod=readonly $(BUILD_FLAGS) -o build/smartbchd ./cmd/smartbchd
# endif

# build-linux: go.sum
# 	GOOS=linux GOARCH=amd64 $(MAKE) build

# .PHONY: all build build-linux

init:
	bash init.sh

up:
	docker-compose up -d

down:
	docker-compose down

reset: down up
