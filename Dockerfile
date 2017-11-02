FROM quay.io/nordstrom/baseimage-ubuntu:16.04

ARG CURRENT_TAG=v0.0.4
ARG KUBELOGIN_DOWNLOAD_BASE_URL=https://github.com/Nordstrom/kubelogin/releases/download

RUN curl -sLo /tmp/kubelogin-server-linux-${CURRENT_TAG}.tar.gz ${KUBELOGIN_DOWNLOAD_BASE_URL}/${CURRENT_TAG}/kubelogin-server-${CURRENT_TAG}-linux.tar.gz \
 && tar -C /tmp/ -xvzf /tmp/kubelogin-server-linux-${CURRENT_TAG}.tar.gz \
 && mv /tmp/kubelogin-server /kubelogin-server

RUN mkdir -p /download/linux /download/mac /download/windows
RUN curl -sLo /download/linux/kubelogin-cli-linux.tar.gz ${KUBELOGIN_DOWNLOAD_BASE_URL}/${CURRENT_TAG}/kubelogin-cli-${CURRENT_TAG}-linux.tar.gz
RUN curl -sLo /download/mac/kubelogin-cli-darwin.tar.gz ${KUBELOGIN_DOWNLOAD_BASE_URL}/${CURRENT_TAG}/kubelogin-cli-${CURRENT_TAG}-darwin.tar.gz
RUN curl -sLo /download/windows/kubelogin-cli-windows.zip ${KUBELOGIN_DOWNLOAD_BASE_URL}/${CURRENT_TAG}/kubelogin-cli-${CURRENT_TAG}-windows.zip

ENTRYPOINT /kubelogin-server
