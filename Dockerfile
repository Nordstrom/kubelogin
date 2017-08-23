FROM quay.io/nordstrom/baseimage-ubuntu:16.04
COPY build/kubelogin /kubelogin
COPY build/kubelogin-cli-windows /
COPY build/kubelogin-cli-linux /
COPY build/kubelogin-cli-darwin /
ENTRYPOINT /kubelogin
