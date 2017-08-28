image_tag := 1.0-g
image_name := quay.io/nordstrom/kubelogin

build:
	mkdir -p build build/download/mac build/download/linux build/download/windows

build/kubelogin : cmd/server/*.go | build
	# Build your golang app for the target OS
	# GOOS=linux GOARCH=amd64 go build -o $@ -ldflags "-X main.Version=$(image_tag)"
	docker run -it \
	  -v $(PWD):/go/src/github.com/nordstrom/kubelogin \
	  -v $(PWD)/build:/go/bin \
	  golang:1.7.4 \
	    go build -v -o /go/bin/kubelogin \
	 	  github.com/nordstrom/kubelogin/cmd/server/

build/kubelogin-cli-% : cmd/cli/*.go | build
	# Build your golang app for the target OS
	# GOOS=linux GOARCH=amd64 go build -o $@ -ldflags "-X main.Version=$(image_tag)"
	docker run -it \
	  -v $(PWD):/go/src/github.com/nordstrom/kubelogin \
	  -v $(PWD)/build:/go/bin \
	  -e GOARCH=amd64 \
	  -e GOOS=$* \
	  golang:1.7.4 \
		go build -v -o /go/bin/kubelogin-cli-$* \
		  github.com/nordstrom/kubelogin/cmd/cli/ \

moveMac:
	cd build/ && mv kubelogin-cli-darwin download/mac/kubelogin

moveLinux:
	cd build/ && mv kubelogin-cli-linux download/linux/kubelogin

moveWindows:
	cd build/ && mv kubelogin-cli-windows download/windows/kubelogin.exe

build/download/mac/kubelogin-cli-darwin.tar.gz: build/kubelogin-cli-darwin moveMac
	cd build/download/mac/ && tar -czf kubelogin-cli-darwin.tar.gz kubelogin

build/download/linux/kubelogin-cli-linux.tar.gz: build/kubelogin-cli-linux moveLinux
	cd build/download/linux/ && tar -czf kubelogin-cli-linux.tar.gz kubelogin

build/download/windows/kubelogin-cli-windows.zip: build/kubelogin-cli-windows moveWindows
	cd build/download/windows/ && zip -r -X kubelogin-cli-windows.zip kubelogin.exe

kubelogin: cmd/server/*.go | build
	# Build golang app for local OS
	go build -o kubelogin

kubeloginCLI: cmd/cli/*.go | build
	go build -o kubeloginCLI

.PHONY: test_app
test_app:
	go test ./...

build/Dockerfile: Dockerfile
	cp Dockerfile build/Dockerfile

tarzip:
	cp build/kubelogin-cli-darwin kubelogin-cli-darwin
	tar -czf kubelogin-cli-darwin.tar.gz kubelogin-cli-darwin
	mv kubelogin-cli-darwin.tar.gz build/

.PHONY: build_image push_image deploy teardown clean
build_image: build/download/linux/kubelogin-cli-linux.tar.gz build/download/windows/kubelogin-cli-windows.zip build/download/mac/kubelogin-cli-darwin.tar.gz build/kubelogin build/Dockerfile | build
	docker build -t $(image_name):$(image_tag) .

push_image: build_image
	docker push $(image_name):$(image_tag)

clean:
	rm -rf build
