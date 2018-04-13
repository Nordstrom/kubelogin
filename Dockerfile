FROM quay.io/nordstrom/baseimage-ubuntu:16.04

ARG CURRENT_TAG=v0.0.0
ARG KUBELOGIN_DOWNLOAD_BASE_URL=https://github.com/Nordstrom/kubelogin/releases/download

COPY build/server/linux/kubelogin-server /kubelogin-server

RUN mkdir -p /download/linux /download/mac /download/windows
COPY build/cli/linux/kubelogin-cli-v0.0.6-dev-linux.tar.gz ${KUBELOGIN_DOWNLOAD_BASE_URL}/${CURRENT_TAG}/kubelogin-cli-${CURRENT_TAG}-linux.tar.gz
COPY build/cli/windows/kubelogin-cli-v0.0.6-dev-windows.zip ${KUBELOGIN_DOWNLOAD_BASE_URL}/${CURRENT_TAG}/kubelogin-cli-${CURRENT_TAG}-windows.zip
COPY build/cli/mac/kubelogin-cli-v0.0.6-dev-darwin.tar.gz ${KUBELOGIN_DOWNLOAD_BASE_URL}/${CURRENT_TAG}/kubelogin-cli-${CURRENT_TAG}-darwin.tar.gz

ENTRYPOINT /kubelogin-server
