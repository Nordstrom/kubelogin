GITHUB_REPO_OWNER := Nordstrom
GITHUB_REPO_NAME := kubelogin
GITHUB_RELEASE_PRERELASE_BOOLEAN := true
GITHUB_RELEASES_UI_URL := https://github.com/$(GITHUB_REPO_OWNER)/$(GITHUB_REPO_NAME)/releases
GITHUB_RELEASES_API_URL := https://api.github.com/repos/$(GITHUB_REPO_OWNER)/$(GITHUB_REPO_NAME)/releases
GITHUB_INDIVIDUAL_RELEASE_ASSET_URL := https://uploads.github.com/repos/$(GITHUB_REPO_OWNER)/$(GITHUB_REPO_NAME)/releases
GITHUB_REPO_HOST_AND_PATH := github.com/$(GITHUB_REPO_OWNER)/$(GITHUB_REPO_NAME)
IMAGE_NAME := quay.io/nordstrom/kubelogin
BUILD := build
CURRENT_TAG := v0.0.4-pre.2
GOLANG_TOOLCHAIN_VERSION := 1.9.1

.PHONY: image/build image/push
.PHONY: release/tag/local release/tag/push
.PHONY: release/github/draft release/github/publish
.PHONY: release/github/draft/create
.PHONY: clean

image/push: image/build
	docker push $(IMAGE_NAME):$(CURRENT_TAG)

# image/build: $(BUILD)/cli/mac/kubelogin-cli-$(CURRENT_TAG)-darwin.tar.gz
# image/build: $(BUILD)/cli/linux/kubelogin-cli-$(CURRENT_TAG)-linux.tar.gz
# image/build: $(BUILD)/cli/windows/kubelogin-cli-$(CURRENT_TAG)-windows-$(CURRENT_TAG).zip
# image/build: $(BUILD)/kubelogin-server-$(CURRENT_TAG)-linux
image/build: $(BUILD)/Dockerfile release/github/publish
	docker build --tag $(IMAGE_NAME):$(CURRENT_TAG) $(<D)

release/github/draft: release/tag/local release/tag/push release/github/draft/create release/assets $(BUILD)/github-release-$(CURRENT_TAG)-id
	@echo "\n\nPlease inspect the release and run make release/github/publish if it looks good"
	open "$(GITHUB_RELEASES_UI_URL)/$(CURRENT_TAG)"

release/github/publish: $(BUILD)/github-release-$(CURRENT_TAG)-published-response-body.json

$(BUILD)/github-release-$(CURRENT_TAG)-published-response-body.json: $(BUILD)/github-release-$(CURRENT_TAG)-published-request-body.json $(BUILD)/github-release-$(CURRENT_TAG)-id
	curl -u $(GITHUB_USERNAME) -XPATCH -d@"$<" "$(GITHUB_RELEASES_API_URL)/$$(cat $(BUILD)/github-release-$(CURRENT_TAG)-id)"

$(BUILD)/github-release-$(CURRENT_TAG)-published-request-body.json: $(BUILD)/github-release-$(CURRENT_TAG)-draft-request-body.json
	jq '.draft = false' "$<" > "$@"

release/tag/local:
	@if [ -n "$$(git status --porcelain)" ]; then echo "Won't tag a dirty working copy. Commit or stash and try again."; exit 1; fi
	-git tag $(CURRENT_TAG)

release/tag/push:
	git push --tags

release/github/draft/create: $(BUILD)/github-release-$(CURRENT_TAG)-response-body.json
release/assets: release/assets/cli/darwin release/assets/cli/linux release/assets/cli/windows

release/assets/cli/darwin: $(BUILD)/cli/mac/kubelogin-cli-$(CURRENT_TAG)-darwin.tar.gz $(BUILD)/github-release-$(CURRENT_TAG)-id
release/assets/cli/linux: $(BUILD)/cli/linux/kubelogin-cli-$(CURRENT_TAG)-linux.tar.gz $(BUILD)/github-release-$(CURRENT_TAG)-id
release/assets/cli/windows: $(BUILD)/cli/windows/kubelogin-cli-$(CURRENT_TAG)-windows.zip $(BUILD)/github-release-$(CURRENT_TAG)-id
release/assets/cli/darwin release/assets/cli/linux release/assets/cli/windows: |
	@if [ -z "$(GITHUB_USERNAME)" ]; then echo "Please set GITHUB_USERNAME"; exit 1; fi
	curl -u $(GITHUB_USERNAME) \
	    --data-binary @"$<" \
	    -H "Content-Type: application/octet-stream" \
	    "$(GITHUB_INDIVIDUAL_RELEASE_ASSET_URL)/$$(cat $(BUILD)/github-release-$(CURRENT_TAG)-id)/assets?name=$(<F)"

release/assets/server/linux: $(BUILD)/kubelogin-server-$(CURRENT_TAG)-linux.tar.gz $(BUILD)/github-release-$(CURRENT_TAG)-id
release/assets/server: |
	@if [ -z "$(GITHUB_USERNAME)" ]; then echo "Please set GITHUB_USERNAME"; exit 1; fi
	curl -u $(GITHUB_USERNAME) \
	    --data-binary @"$<" \
	    -H "Content-Type: application/octet-stream" \
	    "$(GITHUB_INDIVIDUAL_RELEASE_ASSET_URL)/$$(cat $(BUILD)/github-release-$(CURRENT_TAG)-id)/assets?name=$(<F)"

$(BUILD)/github-release-$(CURRENT_TAG)-id: $(BUILD)/github-release-$(CURRENT_TAG)-response-body.json
	jq -r '.id' "$<" > "$@"

# find an existing, non-draft GH release or create one for the tag $(CURRENT_TAG)
$(BUILD)/github-release-$(CURRENT_TAG)-response-body.json: | $(BUILD)
	curl -s "$(GITHUB_RELEASES_API_URL)/tags/$(CURRENT_TAG)" > "$@"
	if [[ $$(jq -e 'has("id") | not' "$@") ]]; then \
	    make "$(BUILD)/github-release-$(CURRENT_TAG)-response-body-new.json"; \
	    mv "$(BUILD)/github-release-$(CURRENT_TAG)-response-body-new.json" "$@"; \
	fi

$(BUILD)/github-release-$(CURRENT_TAG)-response-body-new.json: $(BUILD)/github-release-$(CURRENT_TAG)-draft-request-body.json release/tag/push
	@if [ -z "$(GITHUB_USERNAME)" ]; then echo "Please set GITHUB_USERNAME"; exit 1; fi
	curl -u $(GITHUB_USERNAME) -XPOST -d@"$<" "$(GITHUB_RELEASES_API_URL)" > "$@"

$(BUILD)/github-release-$(CURRENT_TAG)-draft-request-body.json: Makefile | $(BUILD)
	jq -n '{ \
	  tag_name: "$(CURRENT_TAG)", \
	  target_commitish: "master", \
	  name: "$(CURRENT_TAG)", \
	  body: "kubelogin release $(CURRENT_TAG)", \
	  draft: true, \
	  prerelease: $(GITHUB_RELEASE_PRERELASE_BOOLEAN) \
	}' > "$@"

$(BUILD)/kubelogin-server-$(CURRENT_TAG)-linux.tar.gz: $(BUILD)/kubelogin-server-$(CURRENT_TAG)-linux
$(BUILD)/cli/mac/kubelogin-cli-$(CURRENT_TAG)-darwin.tar.gz: $(BUILD)/cli/mac/kubelogin
$(BUILD)/cli/linux/kubelogin-cli-$(CURRENT_TAG)-linux.tar.gz: $(BUILD)/cli/linux/kubelogin
$(BUILD)/cli/mac/kubelogin-cli-$(CURRENT_TAG)-darwin.tar.gz $(BUILD)/cli/linux/kubelogin-cli-$(CURRENT_TAG)-linux.tar.gz:
	tar -C "$(@D)" -czf $@ "$(<F)"

$(BUILD)/cli/windows/kubelogin-cli-$(CURRENT_TAG)-windows-$(CURRENT_TAG).zip: $(BUILD)/cli/windows/kubelogin.exe
	cd "$(@D)" && zip -r -X "$(@F)" "$(<F)"

$(BUILD)/cli/mac/kubelogin: $(BUILD)/kubelogin-cli-$(CURRENT_TAG)-darwin | $(BUILD)/cli/mac
$(BUILD)/cli/linux/kubelogin: $(BUILD)/kubelogin-cli-$(CURRENT_TAG)-linux | $(BUILD)/cli/linux
$(BUILD)/cli/windows/kubelogin.exe: $(BUILD)/kubelogin-cli-$(CURRENT_TAG)-windows | $(BUILD)/cli/windows
$(BUILD)/kubelogin-server: $(BUILD)/kubelogin-server-$(CURRENT_TAG)-darwin
$(BUILD)/kubelogin-cli: $(BUILD)/kubelogin-cli-$(CURRENT_TAG)-darwin
$(BUILD)/Dockerfile: Dockerfile
$(BUILD)/cli/mac/kubelogin $(BUILD)/cli/linux/kubelogin $(BUILD)/cli/windows/kubelogin.exe $(BUILD)/kubelogin-server $(BUILD)/kubelogin-cli $(BUILD)/Dockerfile: | build
	cp "$<" "$@"

# Build your golang app for the target OS
# GOOS=linux GOARCH=amd64 go build -o $@ -ldflags "-X main.Version=$(CURRENT_TAG)"
$(BUILD)/kubelogin-server-$(CURRENT_TAG)-% : cmd/server/*.go | build
	docker run -it \
	  -v $(PWD):/go/src/$(GITHUB_REPO_HOST_AND_PATH) \
	  -v $(PWD)/build:/go/bin \
	  -e GOARCH=amd64 \
	  -e GOOS=$* \
	  golang:$(GOLANG_TOOLCHAIN_VERSION) \
	    go build -v -o /go/bin/$(F@) \
	      $(GITHUB_REPO_HOST_AND_PATH)/cmd/server/

$(BUILD)/kubelogin-cli-$(CURRENT_TAG)-% : cmd/cli/*.go | build
	docker run -it \
	  -v $(PWD):/go/src/$(GITHUB_REPO_HOST_AND_PATH) \
	  -v $(PWD)/build:/go/bin \
	  -e GOARCH=amd64 \
	  -e GOOS=$* \
	  golang:$(GOLANG_TOOLCHAIN_VERSION) \
	    go build -v -o /go/bin/$(F@) \
	      $(GITHUB_REPO_HOST_AND_PATH)/cmd/cli/ \

.PHONY: test_app
test_app:
	go test ./...

build $(BUILD)/cli/mac $(BUILD)/cli/linux $(BUILD)/cli/windows:
	mkdir -p $@

clean:
	rm -rf build
