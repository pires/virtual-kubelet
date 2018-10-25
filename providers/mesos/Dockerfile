FROM ubuntu:18.04

ENV GOLANG_VERSION 1.11.2
ENV GOPATH=/go

ARG KUBERNETES_VERSION=v1.12.2

ENV KUBERNETES_VERSION ${KUBERNETES_VERSION}
ENV KUBECTL_URL https://storage.googleapis.com/kubernetes-release/release/$KUBERNETES_VERSION/bin/linux/amd64/kubectl

ENV PATH "${PATH}:${GOPATH}/bin:/usr/local/go/bin"

# Install needed packages
RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y \
  curl \
  make \
  build-essential \
  git \
  locales \
  sudo \
  net-tools \
  dnsutils \
  bash-completion ; \
  apt-get autoclean; \
  rm -rf /var/lib/apt/lists/*

# Set UTF8 locales
RUN sed -i -e 's/# en_US.UTF-8 UTF-8/en_US.UTF-8 UTF-8/' /etc/locale.gen ; \
    dpkg-reconfigure --frontend=noninteractive locales ; \
    update-locale LANG=en_US.UTF-8

# these variables can only be set after locales are installed and generated
# otherwise we will get warnings in build-time and run-time
ENV LC_ALL en_US.UTF-8
ENV LANG en_US.UTF-8

# Enable bash-completion
RUN echo "source /etc/bash_completion" >> /root/.bashrc

# Setup our path so that the go, dep and kubetest binaries can be found
ENV PATH "${PATH}:${GOPATH}/bin:/usr/local/go/bin"

# Install go and dep
RUN curl -fsSL -o /tmp/go.tar.gz "https://dl.google.com/go/go$GOLANG_VERSION.linux-amd64.tar.gz" ; \
    tar -C /usr/local -xzf /tmp/go.tar.gz ; \
    rm -f /tmp/go.tar.gz ; \
    mkdir -p "$GOPATH/src" "$GOPATH/bin" ; \
    chmod -R 777 "$GOPATH" ; \
    go get -u github.com/golang/dep/cmd/dep

# Install kubectl (for debug purposes alone)
RUN curl -fsSL "$KUBECTL_URL" -o kubectl ; \
    echo "$(curl -fsSL ${KUBECTL_URL}.sha1) kubectl" | sha1sum -c - ; \
    chmod +x kubectl

WORKDIR $GOPATH/src/github.com/virtual-kubelet/virtual-kubelet