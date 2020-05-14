ARG GOLANG

FROM ${GOLANG}

ARG WORKDIR

ENV CGO_ENABLED=0

WORKDIR ${WORKDIR}

COPY . .

RUN \
  make mock && go test -timeout 15s -v -coverprofile=coverage.txt -covermode=atomic ./... && \
  make lint && \
  make build-lambda-api