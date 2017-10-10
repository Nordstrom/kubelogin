FROM quay.io/nordstrom/baseimage-ubuntu:16.04

ARG CURRENT_TAG=v0.0.4-pre.2
ARG KUBELOGIN_DOWNLOAD_BASE_URL=https://github.com/Nordstrom/kubelogin/releases/download

RUN curl -sLo ${KUBELOGIN_DOWNLOAD_BASE_URL}/${CURRENT_TAG}/kubelogin-server-linux-${CURRENT_TAG}.tar.gz /tmp/ \
 && tar -xvzf /tmp/kubelogin-server-linux-${CURRENT_TAG}.tar.gz \
 && mv /tmp/kubelogin /kubelogin

RUN curl -sLo ${KUBELOGIN_DOWNLOAD_BASE_URL}/${CURRENT_TAG}/kubelogin-cli-linux-${CURRENT_TAG}.tar.gz /download/linux/kubelogin-cli-linux.tar.gz
RUN curl -sLo ${KUBELOGIN_DOWNLOAD_BASE_URL}/${CURRENT_TAG}/kubelogin-cli-darwin-${CURRENT_TAG}.tar.gz /download/mac/kubelogin-cli-darwin.tar.gz
RUN curl -sLo ${KUBELOGIN_DOWNLOAD_BASE_URL}/${CURRENT_TAG}/kubelogin-cli-windows-${CURRENT_TAG}.zip /download/mac/kubelogin-cli-windows.zip

ENTRYPOINT /kubelogin
