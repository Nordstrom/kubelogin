GITHUB_REPO_OWNER := Nordstrom
GITHUB_REPO_NAME := kubelogin
GITHUB_RELEASE_PRERELASE_BOOLEAN := true
GITHUB_RELEASES_URL := https://api.github.com/repos/$(GITHUB_REPO_OWNER)/$(GITHUB_REPO_NAME)/releases
GITHUB_INDIVIDUAL_RELEASE_ASSET_URL := https://uploads.github.com/repos/$(GITHUB_REPO_OWNER)/$(GITHUB_REPO_NAME)/releases
GITHUB_REPO_HOST_AND_PATH := github.com/$(GITHUB_REPO_OWNER)/$(GITHUB_REPO_NAME)
IMAGE_NAME := quay.io/nordstrom/kubelogin
BUILD := build
CURRENT_TAG := v0.0.3

.PHONY: image/build image/push
.PHONY: release/tag/local release/tag/push
.PHONY: release/github/create release/github/update
.PHONY: clean

image/push: image/build
	docker push $(IMAGE_NAME):$(CURRENT_TAG)

image/build: $(BUILD)/cli/mac/kubelogin-cli-darwin-$(CURRENT_TAG).tar.gz
image/build: $(BUILD)/cli/linux/kubelogin-cli-linux-$(CURRENT_TAG).tar.gz
image/build: $(BUILD)/cli/windows/kubelogin-cli-windows-$(CURRENT_TAG).zip
image/build: $(BUILD)/Dockerfile $(BUILD)/server/kubelogin
	docker build --build-arg CURRENT_TAG=$(CURRENT_TAG) --tag $(IMAGE_NAME):$(CURRENT_TAG) $(<D)

release/github: release/tag/local release/tag/push release/github/create release/assets

release/tag/local:
	git tag $(CURRENT_TAG)

release/tag/push:
	git push --tags

release/github/create: $(BUILD)/github-release-$(CURRENT_TAG)-response-body.json
release/assets: release/assets/darwin release/assets/linux release/assets/windows

release/assets/darwin: $(BUILD)/cli/mac/kubelogin-cli-darwin-$(CURRENT_TAG).tar.gz $(BUILD)/github-release-$(CURRENT_TAG)-id
release/assets/linux: $(BUILD)/cli/linux/kubelogin-cli-linux-$(CURRENT_TAG).tar.gz $(BUILD)/github-release-$(CURRENT_TAG)-id
release/assets/windows: $(BUILD)/cli/windows/kubelogin-cli-windows-$(CURRENT_TAG).zip $(BUILD)/github-release-$(CURRENT_TAG)-id
release/assets/darwin release/assets/linux release/assets/windows: |
	@if [ -z "$(GITHUB_USERNAME)" ]; then echo "Please set GITHUB_USERNAME"; exit 1; fi
	curl -u $(GITHUB_USERNAME) \
	    --data-binary @"$<" \
	    -H "Content-Type: application/octet-stream" \
	    "$(GITHUB_INDIVIDUAL_RELEASE_ASSET_URL)/$$(cat $(BUILD)/github-release-$(CURRENT_TAG)-id)/assets?name=$(<F)"

# find an existing, non-draft GH release or create one for the tag $(CURRENT_TAG)
$(BUILD)/github-release-$(CURRENT_TAG)-response-body.json: | $(BUILD)
	curl -s "$(GITHUB_RELEASES_URL)/tags/$(CURRENT_TAG)" > "$@"
	@if [ -z "$$(cat '$@')" ]; then \
	    make "$(BUILD)/github-release-$(CURRENT_TAG)-response-body-new.json"; \
	    mv "$(BUILD)/github-release-$(CURRENT_TAG)-response-body-new.json" "$@"; \
	fi

$(BUILD)/github-release-$(CURRENT_TAG)-response-body-new.json: $(BUILD)/github-release-$(CURRENT_TAG)-request-body.json
	@if [ -z "$(GITHUB_USERNAME)" ]; then echo "Please set GITHUB_USERNAME"; exit 1; fi
	curl -u $(GITHUB_USERNAME) -XPOST -d@"$<" "$(GITHUB_RELEASES_URL)" > "$@"

$(BUILD)/github-release-$(CURRENT_TAG)-id: $(BUILD)/github-release-$(CURRENT_TAG)-response-body.json
	jq -r '.id' "$<" > "$@"

$(BUILD)/github-release-$(CURRENT_TAG)-request-body.json: Makefile | $(BUILD)
	jq -n '{ \
	  tag_name: "$(CURRENT_TAG)", \
	  target_commitish: "master", \
	  name: "$(CURRENT_TAG)", \
	  body: "kubelogin release $(CURRENT_TAG)", \
	  draft: true, \
	  prerelease: $(GITHUB_RELEASE_PRERELASE_BOOLEAN) \
	}' > "$@"

$(BUILD)/cli/mac/kubelogin-cli-darwin-$(CURRENT_TAG).tar.gz: $(BUILD)/cli/mac/kubelogin
$(BUILD)/cli/linux/kubelogin-cli-linux-$(CURRENT_TAG).tar.gz: $(BUILD)/cli/linux/kubelogin
$(BUILD)/cli/mac/kubelogin-cli-darwin-$(CURRENT_TAG).tar.gz $(BUILD)/cli/linux/kubelogin-cli-linux-$(CURRENT_TAG).tar.gz:
	tar -C "$(@D)" -czf $@ "$(<F)"

$(BUILD)/cli/windows/kubelogin-cli-windows-$(CURRENT_TAG).zip: $(BUILD)/cli/windows/kubelogin.exe
	cd "$(@D)" && zip -r -X "$(@F)" "$(<F)"

$(BUILD)/cli/mac/kubelogin: $(BUILD)/kubelogin-cli-darwin | $(BUILD)/cli/mac
$(BUILD)/cli/linux/kubelogin: $(BUILD)/kubelogin-cli-linux | $(BUILD)/cli/linux
$(BUILD)/cli/windows/kubelogin.exe: $(BUILD)/kubelogin-cli-windows | $(BUILD)/cli/windows
$(BUILD)/cli/mac/kubelogin $(BUILD)/cli/linux/kubelogin $(BUILD)/cli/windows/kubelogin.exe:
	cp $< $@

# Build your golang app for the target OS
# GOOS=linux GOARCH=amd64 go build -o $@ -ldflags "-X main.Version=$(CURRENT_TAG)"
$(BUILD)/server/kubelogin : cmd/server/*.go | build
	docker run -it \
	  -v $(PWD):/go/src/$(GITHUB_REPO_HOST_AND_PATH) \
	  -v $(PWD)/build:/go/bin \
	  golang:1.7.4 \
	    go build -v -o /go/bin/kubelogin \
	      $(GITHUB_REPO_HOST_AND_PATH)/cmd/server/

$(BUILD)/kubelogin-cli-% : cmd/cli/*.go | build
	docker run -it \
	  -v $(PWD):/go/src/$(GITHUB_REPO_HOST_AND_PATH) \
	  -v $(PWD)/build:/go/bin \
	  -e GOARCH=amd64 \
	  -e GOOS=$* \
	  golang:1.9.0 \
	    go build -v -o /go/bin/kubelogin-cli-$* \
	      $(GITHUB_REPO_HOST_AND_PATH)/cmd/cli/ \

# Build golang app for local OS
kubelogin: cmd/server/*.go | build
	go build -o kubelogin ./cmd/server

kubeloginCLI: cmd/cli/*.go | build
	go build -o kubeloginCLI ./cmd/cli

.PHONY: test_app
test_app:
	go test ./...

$(BUILD)/Dockerfile: Dockerfile | build
	cp Dockerfile $(BUILD)/Dockerfile

build $(BUILD)/cli/mac $(BUILD)/cli/linux $(BUILD)/cli/windows:
	mkdir -p $@

clean:
	rm -rf build
