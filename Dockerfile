FROM quay.io/nordstrom/baseimage-ubuntu:16.04
COPY build/kubelogin /kubelogin
COPY build/download/windows/kubelogin-cli-windows.zip /download/windows/
COPY build/download/linux/kubelogin-cli-linux.tar.gz /download/linux/
COPY build/download/mac/kubelogin-cli-darwin.tar.gz /download/mac/
ENTRYPOINT /kubelogin
