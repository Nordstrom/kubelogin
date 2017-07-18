FROM golang
ENV GOPATH=/go
COPY  ./ /go/src/github.com/nordstrom/kubelogin/
WORKDIR /go/src/github.com/nordstrom/kubelogin/
RUN env
RUN go env
RUN go test -v ./...
