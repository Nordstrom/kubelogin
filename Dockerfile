FROM quay.io/nordstrom/baseimage-ubuntu:16.04

ARG CURRENT_TAG=v0.0.0
ARG KUBELOGIN_DOWNLOAD_BASE_URL=https://github.com/Nordstrom/kubelogin/releases/download

COPY server/linux/kubelogin-server /kubelogin-server

ENTRYPOINT /kubelogin-server
