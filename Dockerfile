FROM quay.io/nordstrom/baseimage-ubuntu:16.04
COPY build/kubelogin /kubelogin
COPY build/windows/kubelogin-cli-windows.zip /windows/
COPY build/linux/kubelogin-cli-linux.tar.gz /linux/
COPY build/mac/kubelogin-cli-darwin.tar.gz /mac/
ENTRYPOINT /kubelogin
