FROM quay.io/nordstrom/baseimage-ubuntu:16.04
COPY build/kubelogin /kubelogin
COPY build/download /download
ENTRYPOINT /kubelogin
