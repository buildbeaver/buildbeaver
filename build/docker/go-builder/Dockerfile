FROM golang:1.19
MAINTAINER NA

RUN apt-get update
RUN apt-get install -y curl unzip postgresql-client

RUN PROTOC_VERSION=3.19.1 && \
    mkdir -p /tmp/protoc && \
    cd /tmp/protoc && \
    curl -LO https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-linux-x86_64.zip && \
    unzip protoc-${PROTOC_VERSION}-linux-x86_64.zip && \
    cp -R bin/* /usr/local/bin/ && \
    cp -R include/* /usr/local/include/ && \
    rm -rf /tmp/protoc

RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.27.1
RUN go install github.com/google/wire/cmd/wire@v0.5.0
RUN go install golang.org/x/tools/cmd/goimports@latest
