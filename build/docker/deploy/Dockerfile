FROM ubuntu:20.04
MAINTAINER NA

ENV TERRAFORM_VERSION="1.3.4"
ENV PACKER_VERSION="1.8.4"
ENV PATH="$PATH:/opt/terraform:/opt/packer"

RUN apt-get update
RUN apt-get install -y wget unzip python3-pip git bc
RUN pip install --upgrade awscli

RUN wget "https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_linux_amd64.zip" && \
    mkdir -p /opt/terraform && \
    unzip "terraform_${TERRAFORM_VERSION}_linux_amd64.zip" -d /opt/terraform/ && \
    rm -f "terraform_${TERRAFORM_VERSION}_linux_amd64.zip"

RUN wget "https://releases.hashicorp.com/packer/${PACKER_VERSION}/packer_${PACKER_VERSION}_linux_amd64.zip" && \
    mkdir -p /opt/packer && \
    unzip "packer_${PACKER_VERSION}_linux_amd64.zip" -d /opt/packer/ && \
    rm -f "packer_${PACKER_VERSION}_linux_amd64.zip"

ENTRYPOINT ["/bin/bash"]
