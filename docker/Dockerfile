FROM alpine

RUN echo '@edge https://nl.alpinelinux.org/alpine/edge/main' >> /etc/apk/repositories && \
    echo '@community https://nl.alpinelinux.org/alpine/edge/community' >> /etc/apk/repositories && \
    apk upgrade --update && \
    apk add bash curl ca-certificates

ENV HELM_VERSION="v2.4.2"

RUN curl -s https://storage.googleapis.com/kubernetes-helm/helm-${HELM_VERSION}-linux-amd64.tar.gz | tar -xz -C /tmp && mv /tmp/linux-amd64/helm /usr/bin

RUN helm init --client-only && \
    mkdir -v -p /root/.kube

COPY landscaper bootstrap /usr/bin/

CMD bash

WORKDIR /root
