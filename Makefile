image_tag := 1.0-g
image_name := quay.io/nordstrom/kubelogin

build:
	mkdir -p build

build/kubelogin : cmd/server/*.go | build
	# Build your golang app for the target OS
	# GOOS=linux GOARCH=amd64 go build -o $@ -ldflags "-X main.Version=$(image_tag)"
	docker run -it \
	  -v $(PWD):/go/src/github.com/nordstrom/kubelogin \
	  -v $(PWD)/build:/go/bin \
	  golang:1.7.4 \
	    go build -v -o /go/bin/kubelogin \
	 	  github.com/nordstrom/kubelogin/cmd/server/


kubelogin: cmd/server/*.go | build
	# Build golang app for local OS
	go build -o kubelogin

.PHONY: test_app
test_app:
	go test ./...

build/Dockerfile: Dockerfile
	cp Dockerfile build/Dockerfile

.PHONY: build_image push_image deploy teardown clean
build_image: build/kubelogin build/Dockerfile | build
	docker build -t $(image_name):$(image_tag) .

push_image: build_image
	docker push $(image_name):$(image_tag)

clean:
	rm -rf build
