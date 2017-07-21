build := build
image_tag := 1.0
image_name := quay.io/nordstrom/kubelogin

build:
	mkdir -p build

build/kubelogin : server/*.go | build
	# Build your golang app for the target OS
	# GOOS=linux GOARCH=amd64 go build -o $@ -ldflags "-X main.Version=$(image_tag)"
	docker run -it -v $(PWD):/go/src/github.com/nordstorm/kubelogin/ -w /go/src/github.com/nordstorm/kubelogin/server/ golang:1.7.4 go build -v -o kubelogin -ldflags "-X main.Version=$(image_tag)"
	mv server/kubelogin build

kubelogin: server/*.go | build
	# Build golang app for local OS
	go build -o kubelogin -ldflags "-X main.Version=$(image_tag)"

.PHONY: test_app
test_app:
	go test

build/Dockerfile: Dockerfile
	cp Dockerfile build/Dockerfile

.PHONY: build_image push_image deploy teardown clean
build_image: build/kubelogin build/Dockerfile | build
	docker build -t $(image_name):$(image_tag) build/.

push_image: build_image
	docker push $(image_name):$(image_tag)

deploy:
	kubectl apply --record -f k8s-resources

teardown:
	kubectl delete -f k8s-resources
