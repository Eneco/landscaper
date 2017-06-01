FROM alpine:3.5

ENV HELM_VERSION v2.2.0

RUN apk add --no-cache \
  ca-certificates \
  curl

RUN curl -s https://storage.googleapis.com/kubernetes-helm/helm-${HELM_VERSION}-linux-amd64.tar.gz | \
  tar -xz -C /tmp && mv /tmp/linux-amd64/helm /usr/bin

RUN helm init --client-only

COPY landscaper bootstrap /usr/bin/

CMD ["landscaper"]
ENTRYPOINT ["/usr/bin/bootstrap"]
