FROM quay.io/nordstrom/baseimage-ubuntu:16.04

COPY kubelogin /kubelogin
COPY download/windows/kubelogin-cli-windows.zip /download/windows/
COPY download/linux/kubelogin-cli-linux.tar.gz /download/linux/
COPY download/mac/kubelogin-cli-darwin.tar.gz /download/mac/

ENTRYPOINT /kubelogin
