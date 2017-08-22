FROM quay.io/nordstrom/baseimage-ubuntu:16.04
COPY build/kubelogin /kubelogin
ENTRYPOINT /kubelogin
