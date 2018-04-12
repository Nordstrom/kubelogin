GITHUB_REPO_OWNER := Nordstrom
GITHUB_REPO_NAME := kubelogin
GITHUB_RELEASE_PRERELASE_BOOLEAN := true
GITHUB_RELEASES_UI_URL := https://github.com/$(GITHUB_REPO_OWNER)/$(GITHUB_REPO_NAME)/releases
GITHUB_RELEASES_API_URL := https://api.github.com/repos/$(GITHUB_REPO_OWNER)/$(GITHUB_REPO_NAME)/releases
GITHUB_INDIVIDUAL_RELEASE_ASSET_URL := https://uploads.github.com/repos/$(GITHUB_REPO_OWNER)/$(GITHUB_REPO_NAME)/releases
GITHUB_REPO_HOST_AND_PATH := github.com/$(GITHUB_REPO_OWNER)/$(GITHUB_REPO_NAME)
IMAGE_NAME := quay.io/nordstrom/kubelogin
BUILD := build
CURRENT_BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
CURRENT_TAG := v0.0.55-dev1
GOLANG_TOOLCHAIN_VERSION := 1.9.1

.PHONY: image/build image/push
.PHONY: release/tag/local release/tag/push
.PHONY: release/github/draft release/github/draft/create release/github/publish
.PHONY: release/assets release/assets/cli release/assets/server
.PHONY: release/assets/cli/% release/assets/server/%
.PHONY: clean

image/push: image/build
	docker push $(IMAGE_NAME):$(CURRENT_TAG)

# image/build: $(BUILD)/cli/mac/kubelogin-cli-$(CURRENT_TAG)-darwin.tar.gz
# image/build: $(BUILD)/cli/linux/kubelogin-cli-$(CURRENT_TAG)-linux.tar.gz
# image/build: $(BUILD)/cli/windows/kubelogin-cli-$(CURRENT_TAG)-windows-$(CURRENT_TAG).zip
# image/build: $(BUILD)/server/linux/kubelogin-server-$(CURRENT_TAG)-linux
image/build: $(BUILD)/Dockerfile release/github/publish
	docker build --tag $(IMAGE_NAME):$(CURRENT_TAG) $(<D)

release/github/draft: release/tag/local release/tag/push release/github/draft/create release/assets $(BUILD)/github-release-$(CURRENT_TAG)-id
	@echo "\n\nPlease inspect the release and run make release/github/publish if it looks good"
	open "$(GITHUB_RELEASES_UI_URL)/$(CURRENT_TAG)"

release/github/publish: release/github/draft $(BUILD)/github-release-$(CURRENT_TAG)-published-response-body.json

$(BUILD)/github-release-$(CURRENT_TAG)-published-response-body.json: $(BUILD)/github-release-$(CURRENT_TAG)-published-request-body.json $(BUILD)/github-release-$(CURRENT_TAG)-id
	curl -u $(GITHUB_USERNAME) -XPATCH -d@"$<" "$(GITHUB_RELEASES_API_URL)/$$(cat $(BUILD)/github-release-$(CURRENT_TAG)-id)" > $@

$(BUILD)/github-release-$(CURRENT_TAG)-published-request-body.json: $(BUILD)/github-release-$(CURRENT_TAG)-draft-request-body.json
	jq '.draft = false' "$<" > "$@"

release/tag/local:
	@if [ -n "$$(git status --porcelain)" ]; then echo "Won't tag a dirty working copy. Commit or stash and try again."; exit 1; fi
	-git tag $(CURRENT_TAG)

release/tag/push: $(BUILD)/tags-$(CURRENT_TAG)-pushed

$(BUILD)/tags-$(CURRENT_TAG)-pushed: | $(BUILD)
	git push --tags > "$@"

release/github/draft/create: $(BUILD)/github-release-$(CURRENT_TAG)-response-body.json

RELEASE_ASSETS_CLI_TARGETS += $(BUILD)/github-release-asset-response-kubelogin-cli-$(CURRENT_TAG)-darwin.json
RELEASE_ASSETS_CLI_TARGETS += $(BUILD)/github-release-asset-response-kubelogin-cli-$(CURRENT_TAG)-linux.json
RELEASE_ASSETS_CLI_TARGETS += $(BUILD)/github-release-asset-response-kubelogin-cli-$(CURRENT_TAG)-windows.json

RELEASE_ASSETS_SERVER_TARGETS += $(BUILD)/github-release-asset-response-kubelogin-server-$(CURRENT_TAG)-linux.json

release/assets: release/assets/cli release/assets/server
release/assets/cli: $(RELEASE_ASSETS_CLI_TARGETS)
release/assets/cli/%: $(BUILD)/github-release-asset-response-kubelogin-cli-$(CURRENT_TAG)-%-asset-response.json
release/assets/server: $(RELEASE_ASSETS_SERVER_TARGETS)
release/assets/server/%: $(BUILD)/github-release-asset-response-kubelogin-server-$(CURRENT_TAG)-%.json

# release/assets/cli/darwin: $(BUILD)/cli/mac/kubelogin-cli-$(CURRENT_TAG)-darwin.tar.gz $(BUILD)/github-release-$(CURRENT_TAG)-id
# release/assets/cli/linux: $(BUILD)/cli/linux/kubelogin-cli-$(CURRENT_TAG)-linux.tar.gz $(BUILD)/github-release-$(CURRENT_TAG)-id
# release/assets/cli/windows: $(BUILD)/cli/windows/kubelogin-cli-$(CURRENT_TAG)-windows.zip $(BUILD)/github-release-$(CURRENT_TAG)-id
# $(RELEASE_ASSETS_CLI_TARGETS): |
# 	@if [ -z "$(GITHUB_USERNAME)" ]; then echo "Please set GITHUB_USERNAME"; exit 1; fi
# 	curl -u $(GITHUB_USERNAME) \
# 	    --data-binary @"$<" \
# 	    -H "Content-Type: application/octet-stream" \
# 	    "$(GITHUB_INDIVIDUAL_RELEASE_ASSET_URL)/$$(cat $(BUILD)/github-release-$(CURRENT_TAG)-id)/assets?name=$(<F)"

$(BUILD)/github-release-asset-response-kubelogin-cli-$(CURRENT_TAG)-darwin.json: $(BUILD)/cli/mac/kubelogin-cli-$(CURRENT_TAG)-darwin.tar.gz $(BUILD)/github-release-$(CURRENT_TAG)-id
$(BUILD)/github-release-asset-response-kubelogin-cli-$(CURRENT_TAG)-linux.json: $(BUILD)/cli/linux/kubelogin-cli-$(CURRENT_TAG)-linux.tar.gz $(BUILD)/github-release-$(CURRENT_TAG)-id
$(BUILD)/github-release-asset-response-kubelogin-cli-$(CURRENT_TAG)-windows.json: $(BUILD)/cli/windows/kubelogin-cli-$(CURRENT_TAG)-windows.zip $(BUILD)/github-release-$(CURRENT_TAG)-id
$(BUILD)/github-release-asset-response-kubelogin-server-$(CURRENT_TAG)-linux.json: $(BUILD)/server/linux/kubelogin-server-$(CURRENT_TAG)-linux.tar.gz $(BUILD)/github-release-$(CURRENT_TAG)-id
$(RELEASE_ASSETS_CLI_TARGETS) $(RELEASE_ASSETS_SERVER_TARGETS):
	@if [ -z "$(GITHUB_USERNAME)" ]; then echo "Please set GITHUB_USERNAME"; exit 1; fi
	curl -u $(GITHUB_USERNAME) \
	    --data-binary @"$<" \
	    -H "Content-Type: application/octet-stream" \
	    "$(GITHUB_INDIVIDUAL_RELEASE_ASSET_URL)/$$(cat $(BUILD)/github-release-$(CURRENT_TAG)-id)/assets?name=$(<F)" \
	    | jq '.' > "$@"

$(BUILD)/github-release-$(CURRENT_TAG)-id: $(BUILD)/github-release-$(CURRENT_TAG)-response-body.json
	jq -r '.id' "$<" > "$@"

# find an existing, non-draft GH release or create one for the tag $(CURRENT_TAG)
$(BUILD)/github-release-$(CURRENT_TAG)-response-body.json: | $(BUILD)
	curl -s "$(GITHUB_RELEASES_API_URL)/tags/$(CURRENT_TAG)" > "$@"
	if [[ $$(jq -e 'has("id") | not' "$@") ]]; then \
	    make "$(BUILD)/github-release-$(CURRENT_TAG)-draft-response-body-new.json"; \
	    cp "$(BUILD)/github-release-$(CURRENT_TAG)-draft-response-body-new.json" "$@"; \
	fi

$(BUILD)/github-release-$(CURRENT_TAG)-draft-response-body-new.json: $(BUILD)/github-release-$(CURRENT_TAG)-draft-request-body.json release/tag/push
	@if [ -z "$(GITHUB_USERNAME)" ]; then echo "Please set GITHUB_USERNAME"; exit 1; fi
	curl -u $(GITHUB_USERNAME) -XPOST -d@"$<" "$(GITHUB_RELEASES_API_URL)" > "$@"

$(BUILD)/github-release-$(CURRENT_TAG)-draft-request-body.json: Makefile | $(BUILD)
	jq -n '{ \
	  tag_name: "$(CURRENT_TAG)", \
	  target_commitish: "$(CURRENT_BRANCH)", \
	  name: "$(CURRENT_TAG)", \
	  body: "kubelogin release $(CURRENT_TAG)", \
	  draft: true, \
	  prerelease: $(GITHUB_RELEASE_PRERELASE_BOOLEAN) \
	}' > "$@"

$(BUILD)/cli/mac/kubelogin-cli-$(CURRENT_TAG)-darwin.tar.gz: $(BUILD)/cli/mac/kubelogin
$(BUILD)/cli/linux/kubelogin-cli-$(CURRENT_TAG)-linux.tar.gz: $(BUILD)/cli/linux/kubelogin
$(BUILD)/cli/mac/kubelogin-cli-$(CURRENT_TAG)-darwin.tar.gz $(BUILD)/cli/linux/kubelogin-cli-$(CURRENT_TAG)-linux.tar.gz:
	tar -C "$(@D)" -czf $@ "$(<F)"

$(BUILD)/server/linux/kubelogin-server-$(CURRENT_TAG)-linux.tar.gz: $(BUILD)/server/linux/kubelogin-server
	tar -C "$(@D)" -czf $@ "$(<F)"

$(BUILD)/cli/windows/kubelogin-cli-$(CURRENT_TAG)-windows.zip: $(BUILD)/cli/windows/kubelogin.exe
	cd "$(@D)" && zip -r -X "$(@F)" "$(<F)"

$(BUILD)/cli/mac/kubelogin: $(BUILD)/cli/kubelogin-cli-$(CURRENT_TAG)-darwin | $(BUILD)/cli/mac
$(BUILD)/cli/linux/kubelogin: $(BUILD)/cli/kubelogin-cli-$(CURRENT_TAG)-linux | $(BUILD)/cli/linux
$(BUILD)/cli/windows/kubelogin.exe: $(BUILD)/cli/kubelogin-cli-$(CURRENT_TAG)-windows | $(BUILD)/cli/windows
$(BUILD)/server/linux/kubelogin-server: $(BUILD)/server/kubelogin-server-$(CURRENT_TAG)-linux | $(BUILD)/server/linux
$(BUILD)/Dockerfile: Dockerfile | $(BUILD)
$(BUILD)/cli/mac/kubelogin $(BUILD)/cli/linux/kubelogin $(BUILD)/cli/windows/kubelogin.exe $(BUILD)/server/linux/kubelogin-server $(BUILD)/Dockerfile:
	cp "$<" "$@"

# Build your golang app for the target OS
# GOOS=linux GOARCH=amd64 go build -o $@ -ldflags "-X main.Version=$(CURRENT_TAG)"
$(BUILD)/server/kubelogin-server-$(CURRENT_TAG)-%: cmd/server/*.go | $(BUILD)/server
	docker run -it \
	  -v $(PWD):/go/src/$(GITHUB_REPO_HOST_AND_PATH) \
	  -v $(PWD)/$(@D):/go/bin \
	  -e GOARCH=amd64 \
	  -e GOOS=$* \
	  golang:$(GOLANG_TOOLCHAIN_VERSION) \
	    go build -v -o /go/bin/$(@F) \
	      $(GITHUB_REPO_HOST_AND_PATH)/cmd/server/

$(BUILD)/cli/kubelogin-cli-$(CURRENT_TAG)-%: cmd/cli/*.go | $(BUILD)/cli
	docker run -it \
	  -v $(PWD):/go/src/$(GITHUB_REPO_HOST_AND_PATH) \
	  -v $(PWD)/$(@D):/go/bin \
	  -e GOARCH=amd64 \
	  -e GOOS=$* \
	  golang:$(GOLANG_TOOLCHAIN_VERSION) \
	    go build -v -o /go/bin/$(@F) \
	      $(GITHUB_REPO_HOST_AND_PATH)/cmd/cli/ \

.PHONY: test_app
test_app:
	go test ./...

build $(BUILD)/server $(BUILD)/server/linux $(BUILD)/cli $(BUILD)/cli/mac $(BUILD)/cli/linux $(BUILD)/cli/windows:
	mkdir -p $@

clean:
	rm -rf build
