FROM quay.io/nordstrom/baseimage-ubuntu:16.04

ARG CURRENT_TAG

COPY kubelogin /kubelogin
COPY cli/windows/kubelogin-cli-windows-${CURRENT_TAG}.zip /download/windows/kubelogin-cli-windows.zip
COPY cli/linux/kubelogin-cli-linux-${CURRENT_TAG}.tar.gz /download/linux/kubelogin-cli-linux.tar.gz
COPY cli/mac/kubelogin-cli-darwin-${CURRENT_TAG}.tar.gz /download/mac/kubelogin-cli-darwin.tar.gz

ENTRYPOINT /kubelogin
