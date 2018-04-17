FROM quay.io/nordstrom/baseimage-ubuntu:16.04

ARG CURRENT_TAG=v0.0.0
ARG KUBELOGIN_DOWNLOAD_BASE_URL=https://github.com/Nordstrom/kubelogin/releases/download

COPY server/linux/kubelogin-server /kubelogin-server

RUN mkdir -p /download/linux /download/mac /download/windows
COPY cli/linux/kubelogin-cli-${CURRENT_TAG}-linux.tar.gz ${KUBELOGIN_DOWNLOAD_BASE_URL}/${CURRENT_TAG}/kubelogin-cli-${CURRENT_TAG}-linux.tar.gz
COPY cli/windows/kubelogin-cli-${CURRENT_TAG}-windows.zip ${KUBELOGIN_DOWNLOAD_BASE_URL}/${CURRENT_TAG}/kubelogin-cli-${CURRENT_TAG}-windows.zip
COPY cli/mac/kubelogin-cli-${CURRENT_TAG}-darwin.tar.gz ${KUBELOGIN_DOWNLOAD_BASE_URL}/${CURRENT_TAG}/kubelogin-cli-${CURRENT_TAG}-darwin.tar.gz

ENTRYPOINT /kubelogin-server
