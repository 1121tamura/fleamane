FROM golang:1.23-bookworm

RUN apt-get update && apt-get install -y \
    git \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

RUN groupadd -g 1000 vscode \
    && useradd -u 1000 -g vscode -m -s /bin/bash vscode \
    && echo "vscode ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers

WORKDIR /workspace
USER vscode
