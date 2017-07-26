FROM quay.io/nordstrom/baseimage-ubuntu:16.04
COPY  kubelogin /kubelogin
ENTRYPOINT /kubelogin 
