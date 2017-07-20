build := build
image_tag := 1.0
image_name := quay.io/nordstrom/kubelogin

build:
	mkdir -p build

build/sample-app : *.go | build
	# Build your golang app for the target OS
	# GOOS=linux GOARCH=amd64 go build -o $@ -ldflags "-X main.Version=$(image_tag)"
	docker run -it -v $(PWD):/go/src/sample-app -w /go/src/sample-app golang:1.7.4 go build -v
	mv sample-app build

sample-app: *.go | build
	# Build golang app for local OS
	go build -o sample-app -ldflags "-X main.Version=$(image_tag)"

.PHONY: test_app
test_app:
	go test

build/Dockerfile: Dockerfile
	cp Dockerfile build/Dockerfile

.PHONY: build_image push_image deploy teardown clean
build_image: build/sample-app build/Dockerfile | build
	docker build -t $(image_name):$(image_tag) build

push_image: build_image
	docker push $(image_name):$(image_tag)

deploy:
	kubectl apply --record -f k8s-resources

teardown:
	kubectl delete -f k8s-resources
