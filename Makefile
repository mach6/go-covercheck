APP_MODULE_PATH		?= github.com/mach6/
APP_NAME			?= go-covercheck
APP_REVISION 		:= $(shell git rev-parse --short HEAD)
APP_VERSION 		?= $(shell cat VERSION)
BUILD_TIME_STAMP 	:= $(shell date +%FT%TZ)
BUILT_BY 			?= $(shell whoami)

LD_FLAGS := " -X $(APP_MODULE_PATH)$(APP_NAME)/pkg/config.AppVersion=$(APP_VERSION) \
			  -X $(APP_MODULE_PATH)$(APP_NAME)/pkg/config.AppRevision=$(APP_REVISION) \
			  -X $(APP_MODULE_PATH)$(APP_NAME)/pkg/config.AppName=$(APP_NAME) \
			  -X $(APP_MODULE_PATH)$(APP_NAME)/pkg/config.BuildTimeStamp=$(BUILD_TIME_STAMP) \
			  -X $(APP_MODULE_PATH)$(APP_NAME)/pkg/config.BuiltBy=$(BUILT_BY)"

all: lint test covercheck build dist docker

lint:
	golangci-lint run ./...

test:
	go test -v -coverprofile coverage.out ./...

build:
	go build -trimpath -a -o $(APP_NAME) -ldflags=$(LD_FLAGS) \
		$(APP_MODULE_PATH)$(APP_NAME)/cmd/$(APP_NAME)

covercheck:
	./$(APP_NAME) -C v0.6.0

clean:
	rm -rf dist/ \
	&& rm -f ./$(APP_NAME) \
	&& rm coverage.out

dist:
	mkdir -p dist
	GOOS=linux GOARCH=amd64 \
		go build -trimpath -a -o dist/$(APP_NAME)_linux_amd64 \
			-ldflags=$(LD_FLAGS) $(APP_MODULE_PATH)$(APP_NAME)/cmd/$(APP_NAME)

	GOOS=linux GOARCH=arm64 \
		go build -trimpath -a -o dist/$(APP_NAME)_linux_arm64 \
			-ldflags=$(LD_FLAGS) $(APP_MODULE_PATH)$(APP_NAME)/cmd/$(APP_NAME)

	GOOS=freebsd GOARCH=amd64 \
		go build -trimpath -a -o dist/$(APP_NAME)_freebsd_amd64 \
			-ldflags=$(LD_FLAGS) $(APP_MODULE_PATH)$(APP_NAME)/cmd/$(APP_NAME)

	GOOS=freebsd GOARCH=arm64 \
		go build -trimpath -a -o dist/$(APP_NAME)_freebsd_arm64 \
			-ldflags=$(LD_FLAGS) $(APP_MODULE_PATH)$(APP_NAME)/cmd/$(APP_NAME)

	GOOS=darwin GOARCH=amd64 \
		go build -trimpath -a -o dist/$(APP_NAME)_darwin_amd64 \
			-ldflags=$(LD_FLAGS) $(APP_MODULE_PATH)$(APP_NAME)/cmd/$(APP_NAME)

	GOOS=darwin GOARCH=arm64 \
		go build -trimpath -a -o dist/$(APP_NAME)_darwin_arm64 \
			-ldflags=$(LD_FLAGS) $(APP_MODULE_PATH)$(APP_NAME)/cmd/$(APP_NAME)

	GOOS=windows GOARCH=amd64 \
		go build -trimpath -a -o dist/$(APP_NAME)_windows_amd64.exe \
			--ldflags=$(LD_FLAGS) $(APP_MODULE_PATH)$(APP_NAME)/cmd/$(APP_NAME)

	GOOS=windows GOARCH=arm64 \
		go build -trimpath -a -o dist/$(APP_NAME)_windows_arm64.exe \
			--ldflags=$(LD_FLAGS) $(APP_MODULE_PATH)$(APP_NAME)/cmd/$(APP_NAME)

	strip ./dist/*linux_amd64*

changelog:
	# Generate a changelog for tag and merge it with the existing CHANGELOG.md
	# Yeah this is some ugly shell scripting, but it works.
	@TAG=$(APP_VERSION); \
	git-chglog $$TAG > NEW_TAG_CHANGELOG.md; \
	start=$$(grep -n '^## \[Unreleased\]' CHANGELOG.md | head -n1 | cut -d: -f1); \
	end=$$(grep -n '^## \[' CHANGELOG.md | grep -A1 '^'$$start':' | tail -n1 | cut -d: -f1); \
	if [ -z "$$end" ]; then end=$$(wc -l < CHANGELOG.md); fi; \
	head -n $$((start - 1)) CHANGELOG.md > TMP_CHANGELOG.md; \
	cat NEW_TAG_CHANGELOG.md >> TMP_CHANGELOG.md; \
	tail -n +$$end CHANGELOG.md >> TMP_CHANGELOG.md; \
	mv TMP_CHANGELOG.md CHANGELOG.md; \
	rm -f NEW_TAG_CHANGELOG.md

docker:
	docker build -t $(APP_NAME):$(APP_VERSION) .

.PHONY: clean lint build test covercheck dist docker
