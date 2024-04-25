FROM ubuntu:20.04
MAINTAINER NA

ENV TERRAFORM_VERSION="1.3.4"
ENV PACKER_VERSION="1.8.4"
ENV NODE_VERSION=18.16.0
ENV PATH="$PATH:/opt/terraform:/opt/packer"

RUN apt-get update
RUN apt-get install -y software-properties-common
RUN apt-add-repository ppa:ansible/ansible
RUN apt-get update
RUN apt-get install -y wget curl unzip git bc ansible

# Custom apt repo to install python3.11 which we rely on
RUN add-apt-repository ppa:deadsnakes/ppa
RUN apt-get install -y python3.11
RUN curl -sS https://bootstrap.pypa.io/get-pip.py | python3.11
RUN pip install --upgrade awscli pytest

# Terraform install
RUN wget "https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_linux_amd64.zip" && \
    mkdir -p /opt/terraform && \
    unzip "terraform_${TERRAFORM_VERSION}_linux_amd64.zip" -d /opt/terraform/ && \
    rm -f "terraform_${TERRAFORM_VERSION}_linux_amd64.zip"

# Packer install
RUN wget "https://releases.hashicorp.com/packer/${PACKER_VERSION}/packer_${PACKER_VERSION}_linux_amd64.zip" && \
    mkdir -p /opt/packer && \
    unzip "packer_${PACKER_VERSION}_linux_amd64.zip" -d /opt/packer/ && \
    rm -f "packer_${PACKER_VERSION}_linux_amd64.zip"

# Node install
RUN curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.0/install.sh | bash
ENV NVM_DIR=/root/.nvm
RUN . "$NVM_DIR/nvm.sh" && \
    nvm install ${NODE_VERSION} && \
    nvm use v${NODE_VERSION} && \
    nvm alias default v${NODE_VERSION}
ENV PATH="/root/.nvm/versions/node/v${NODE_VERSION}/bin/:${PATH}"

# Yarn install
RUN corepack enable && \
    corepack prepare yarn@stable --activate

# Docker install
RUN curl -fsSL https://get.docker.com/ | sh

# Add our E2E test runner script
ADD docker/e2e/buildbeaver-e2e-runner.sh /usr/bin/
RUN chmod +x /usr/bin/buildbeaver-e2e-runner.sh

ENTRYPOINT ["/usr/bin/buildbeaver-e2e-runner.sh"]
