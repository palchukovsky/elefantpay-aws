ARG GOLANG_TAG

FROM golang:${GOLANG_TAG}

ARG GOLANGCI_VER
ARG AWSCLI_VERSION
ARG WORKDIR

ENV GO111MODULE=on
ENV GLIBC_VER=2.31-r0

# Installing required tools (system, AWS cli, linter).
RUN \
  apk update && apk add build-base curl zip && \
  curl -sL https://alpine-pkgs.sgerrand.com/sgerrand.rsa.pub -o /etc/apk/keys/sgerrand.rsa.pub && \
  curl -sLO https://github.com/sgerrand/alpine-pkg-glibc/releases/download/${GLIBC_VER}/glibc-${GLIBC_VER}.apk && \
  curl -sLO https://github.com/sgerrand/alpine-pkg-glibc/releases/download/${GLIBC_VER}/glibc-bin-${GLIBC_VER}.apk && \
  apk add --no-cache glibc-${GLIBC_VER}.apk glibc-bin-${GLIBC_VER}.apk && \
  rm glibc-${GLIBC_VER}.apk && \
  rm glibc-bin-${GLIBC_VER}.apk && \
  rm -rf /var/cache/apk/* && \
  curl -sL https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip -o /awscli.zip && \
  unzip -q /awscli.zip -d / && \
  /aws/install && \
  rm -rf /aws /awscli.zip /usr/local/aws-cli/v2/*/dist/aws_completer /usr/local/aws-cli/v2/*/dist/awscli/data/ac.index /usr/local/aws-cli/v2/*/dist/awscli/examples && \
  echo "AWS CLI version: " && aws --version && \
  curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v${GOLANGCI_VER}

# Installing unit tests and project dependencies.
WORKDIR ${WORKDIR}
COPY Makefile go.mod go.sum ./
RUN \
  make install-mock-deps && \
  go mod download